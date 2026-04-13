package poker

import (
	"encoding/json"
	"log"
	"sync"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

const (
	MaxUserNameLength = 30
	MaxUsersPerRoom   = 50
)

// Valid vote values (Fibonacci-like for planning poker)
var ValidVotes = map[int]bool{
	1: true, 2: true, 3: true, 5: true, 8: true, 13: true, 21: true,
}

type Hub struct {
	Rooms map[string]*Room
	mu    sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[string]*Room),
	}
}

func (h *Hub) GetRoom(id string) (*Room, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.Rooms[id]
	return room, ok
}

func (h *Hub) CreateRoom(id string, managerToken string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	room := &Room{
		ID:           id,
		Users:        make(map[string]*User),
		hub:          h,
		ManagerToken: managerToken,
	}
	h.Rooms[id] = room
	return room
}

func (h *Hub) RemoveRoom(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Rooms, id)
}

type User struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	HasVoted  bool            `json:"hasVoted"`
	IsManager bool            `json:"isManager"`
	VoteValue *int            `json:"-"`
	Conn      *websocket.Conn `json:"-"`
}

type Room struct {
	ID           string
	Users        map[string]*User
	mu           sync.Mutex
	Revealed     bool
	hub          *Hub
	ManagerToken string // Secret token given to the room creator
}

type Message struct {
	Action  string          `json:"action"`
	Payload json.RawMessage `json:"payload"`
}

// Client payloads
type JoinRoomPayload struct {
	RoomID       string `json:"roomId"`
	UserName     string `json:"userName"`
	ManagerToken string `json:"managerToken"` // Client sends the secret token to prove manager status
}

type SubmitVotePayload struct {
	VoteValue *int `json:"voteValue"`
}

// Server payloads
type RoomStatePayload struct {
	Players  []*User `json:"players"`
	Revealed bool    `json:"revealed"`
}

type UserUpdatedPayload struct {
	User *User `json:"user"`
}

type JoinedPayload struct {
	UserID string `json:"userId"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type CardsRevealedPayload struct {
	Votes   []UserVote `json:"votes"`
	Average float64    `json:"average"`
}

type UserVote struct {
	UserID    string `json:"userId"`
	VoteValue *int   `json:"voteValue"`
}

// SanitizeUserName trims and limits the username length
func SanitizeUserName(name string) string {
	// Trim whitespace
	trimmed := ""
	start := 0
	end := len(name)
	for start < end && (name[start] == ' ' || name[start] == '\t' || name[start] == '\n' || name[start] == '\r') {
		start++
	}
	for end > start && (name[end-1] == ' ' || name[end-1] == '\t' || name[end-1] == '\n' || name[end-1] == '\r') {
		end--
	}
	trimmed = name[start:end]

	if trimmed == "" {
		return ""
	}

	// Truncate to MaxUserNameLength runes
	if utf8.RuneCountInString(trimmed) > MaxUserNameLength {
		runes := []rune(trimmed)
		trimmed = string(runes[:MaxUserNameLength])
	}

	return trimmed
}

func (r *Room) UserCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Users)
}

func (r *Room) SendToUser(userID string, action string, payload interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Println("Error marshaling payload", err)
		return
	}

	msg := Message{Action: action, Payload: body}
	msgBytes, _ := json.Marshal(msg)

	r.mu.Lock()
	defer r.mu.Unlock()

	if u, ok := r.Users[userID]; ok && u.Conn != nil {
		u.Conn.WriteMessage(websocket.TextMessage, msgBytes)
	}
}

func (r *Room) Broadcast(action string, payload interface{}) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Println("Error marshaling payload", err)
		return
	}

	msg := Message{Action: action, Payload: body}
	msgBytes, _ := json.Marshal(msg)

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.Users {
		if u.Conn != nil {
			err := u.Conn.WriteMessage(websocket.TextMessage, msgBytes)
			if err != nil {
				log.Println("Error writing message", err)
			}
		}
	}
}

func (r *Room) HandleMessage(userID string, msg Message) {
	switch msg.Action {
	case "SUBMIT_VOTE":
		var p SubmitVotePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			log.Println("Invalid vote payload:", err)
			return
		}

		// Validate vote value: must be nil (pass) or a valid Fibonacci value
		if p.VoteValue != nil {
			if !ValidVotes[*p.VoteValue] {
				r.SendToUser(userID, "ERROR", ErrorPayload{Message: "Invalid vote value"})
				return
			}
		}

		r.mu.Lock()
		u, ok := r.Users[userID]
		if ok {
			u.VoteValue = p.VoteValue
			u.HasVoted = true
		}
		revealed := r.Revealed
		r.mu.Unlock()

		if revealed {
			return // Cannot impact state if already revealed
		}

		if ok {
			userUpdated := UserUpdatedPayload{
				User: u,
			}
			r.Broadcast("USER_UPDATED", userUpdated)
		}

	case "REVEAL_CARDS":
		r.mu.Lock()
		u, ok := r.Users[userID]
		if !ok || !u.IsManager {
			r.mu.Unlock()
			return
		}
		r.Revealed = true

		var votes []UserVote
		var sum int
		var count int

		for _, user := range r.Users {
			if user.HasVoted {
				votes = append(votes, UserVote{UserID: user.ID, VoteValue: user.VoteValue})
				if user.VoteValue != nil {
					sum += *user.VoteValue
					count++
				}
			}
		}

		var average float64
		if count > 0 {
			average = float64(sum) / float64(count)
		}
		r.mu.Unlock()

		r.Broadcast("CARDS_REVEALED", CardsRevealedPayload{Votes: votes, Average: average})

	case "RESET_ROOM":
		r.mu.Lock()
		u, exists := r.Users[userID]
		if !exists || !u.IsManager {
			r.mu.Unlock()
			return
		}
		r.Revealed = false
		for _, user := range r.Users {
			user.VoteValue = nil
			user.HasVoted = false
		}
		r.mu.Unlock()

		r.Broadcast("ROOM_RESET", map[string]interface{}{})
		r.BroadcastState()
	}
}

func (r *Room) BroadcastState() {
	r.mu.Lock()
	var players []*User
	for _, u := range r.Users {
		players = append(players, u)
	}
	revealed := r.Revealed
	r.mu.Unlock()

	st := RoomStatePayload{
		Players:  players,
		Revealed: revealed,
	}
	r.Broadcast("ROOM_STATE", st)
}

func (r *Room) AddUser(u *User) {
	r.mu.Lock()
	r.Users[u.ID] = u
	r.mu.Unlock()

	r.BroadcastState()
}

func (r *Room) RemoveUser(userID string) {
	r.mu.Lock()
	if user, ok := r.Users[userID]; ok {
		if user.Conn != nil {
			user.Conn.Close()
		}
		delete(r.Users, userID)
	}
	empty := len(r.Users) == 0
	roomID := r.ID
	r.mu.Unlock()

	// Clean up empty rooms to prevent memory leaks
	if empty {
		r.hub.RemoveRoom(roomID)
		log.Printf("Room %s removed (empty)", roomID)
	} else {
		r.BroadcastState()
	}
}
