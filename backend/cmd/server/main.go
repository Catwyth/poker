package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"poker/internal/poker"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var allowedOrigins map[string]bool

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		allowed := allowedOrigins[origin]
		if !allowed {
			log.Printf("websocket: rejected origin %q (not in allowed list)", origin)
		}
		return allowed
	},
}

var hub *poker.Hub

// Simple in-memory rate limiter
type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{requests: make(map[string][]time.Time)}
}

func (rl *rateLimiter) allow(ip string, maxRequests int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	var recent []time.Time
	for _, t := range rl.requests[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= maxRequests {
		rl.requests[ip] = recent
		return false
	}

	rl.requests[ip] = append(recent, now)
	return true
}

var apiLimiter = newRateLimiter()
var wsLimiter = newRateLimiter()

func main() {
	hub = poker.NewHub()

	// Parse allowed origins from environment
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv == "" {
		originsEnv = "http://localhost:3000"
	}
	allowedOrigins = make(map[string]bool)
	for _, o := range strings.Split(originsEnv, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			allowedOrigins[trimmed] = true
			log.Printf("Allowed origin: %s", trimmed)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/rooms", handleCreateRoom)
	mux.HandleFunc("/ws", handleWebSocket)
	mux.HandleFunc("/health", handleHealth)

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Starting server on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ListenAndServe:", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server stopped.")
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if allowedOrigins[origin] {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Vary", "Origin")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientIP := getClientIP(r)
	if !apiLimiter.allow(clientIP, 10, time.Minute) {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	roomID := uuid.New().String()
	managerToken := uuid.New().String()
	hub.CreateRoom(roomID, managerToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"roomId":       roomID,
		"managerToken": managerToken,
	})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !wsLimiter.allow(clientIP, 20, time.Minute) {
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}

	conn.SetReadLimit(4096)

	// Configure ping/pong heartbeat
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	var userID string
	var myRoom *poker.Room

	// Start ping ticker
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	defer func() {
		close(done)
		if myRoom != nil && userID != "" {
			myRoom.RemoveUser(userID)
		}
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Reset read deadline on each received message
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var msg poker.Message
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("Invalid message format:", err)
			continue
		}

		if msg.Action == "JOIN_ROOM" {
			var p poker.JoinRoomPayload
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				log.Println("Invalid payload for JOIN_ROOM:", err)
				continue
			}

			if !uuidRegex.MatchString(p.RoomID) {
				sendError(conn, "Invalid room ID format")
				continue
			}

			sanitizedName := poker.SanitizeUserName(p.UserName)
			if sanitizedName == "" {
				sendError(conn, "Username is required")
				continue
			}

			room, exists := hub.GetRoom(p.RoomID)
			if !exists {
				sendError(conn, "Room not found")
				continue
			}

			if room.UserCount() >= poker.MaxUsersPerRoom {
				sendError(conn, "Room is full")
				continue
			}

			isManager := p.ManagerToken != "" && p.ManagerToken == room.ManagerToken

			userID = uuid.New().String()
			myRoom = room

			u := &poker.User{
				ID:        userID,
				Name:      sanitizedName,
				HasVoted:  false,
				IsManager: isManager,
				Conn:      conn,
			}
			room.AddUser(u)

			room.SendToUser(userID, "JOINED", poker.JoinedPayload{UserID: userID})
		} else if myRoom != nil {
			myRoom.HandleMessage(userID, msg)
		}
	}
}

func sendError(conn *websocket.Conn, message string) {
	payload, _ := json.Marshal(poker.ErrorPayload{Message: message})
	msg := poker.Message{Action: "ERROR", Payload: payload}
	msgBytes, _ := json.Marshal(msg)
	conn.WriteMessage(websocket.TextMessage, msgBytes)
}
