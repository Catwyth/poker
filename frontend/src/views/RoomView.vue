<template>
  <div class="w-full max-w-5xl flex flex-col gap-8 h-full min-h-[80vh]">
    <!-- Header -->
    <header class="flex flex-col md:flex-row justify-between items-center gap-4 bg-white/5 backdrop-blur-md border border-white/10 p-6 rounded-3xl shadow-xl">
      <div>
        <h1 class="text-2xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-indigo-300 to-purple-300">Room: {{ roomId }}</h1>
        <p class="text-slate-400 text-sm mt-1">Hello, {{ userName }} <span v-if="isManager" class="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-indigo-500/20 text-indigo-300 border border-indigo-500/30">Manager</span></p>
      </div>

      <div class="flex gap-3" v-if="isManager">
        <button @click="resetRoom" class="px-4 py-2 bg-slate-800 hover:bg-slate-700 text-slate-300 rounded-xl transition-colors font-medium border border-slate-700">
          Reset Room
        </button>
        <button @click="revealCards" :disabled="roomState.revealed" class="px-6 py-2 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-xl shadow-[0_0_20px_-5px_rgba(99,102,241,0.5)] transition-all font-bold tracking-wide">
          Reveal Cards
        </button>
      </div>
    </header>

    <!-- Connection status banner -->
    <div v-if="connectionStatus === 'disconnected'" class="bg-red-500/20 border border-red-500/30 text-red-300 px-4 py-3 rounded-xl text-center text-sm font-medium animate-pulse">
      Connection lost. Reconnecting...
    </div>
    <div v-if="connectionStatus === 'error'" class="bg-red-500/20 border border-red-500/30 text-red-300 px-4 py-3 rounded-xl text-center text-sm font-medium">
      {{ errorMessage }}
    </div>

    <div class="flex-1 grid grid-cols-1 lg:grid-cols-3 gap-8">
      <!-- Players Area -->
      <div class="lg:col-span-2 bg-white/5 backdrop-blur-md border border-white/10 rounded-3xl p-8 flex flex-col items-center justify-center relative min-h-[400px]">
        
        <!-- Table -->
        <div class="w-64 h-32 md:w-96 md:h-48 bg-indigo-900/40 rounded-[100px] border border-indigo-500/20 shadow-[0_0_50px_rgba(49,46,129,0.3)] absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 flex items-center justify-center">
             <div v-if="roomState.revealed" class="text-center animate-fade-in">
               <p class="text-slate-400 text-sm font-semibold uppercase tracking-widest mb-1">Average</p>
               <p class="text-5xl font-bold text-white drop-shadow-md">{{ average === 0 ? '-' : average.toFixed(1) }}</p>
             </div>
        </div>
        
        <!-- Players placed parametrically around the table -->
        <div class="w-full h-full relative z-10 grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 gap-6 content-between place-items-center">
            <div v-for="player in roomState.players" :key="player.id" class="flex flex-col items-center group z-10">
               
               <!-- Card -->
               <div class="relative w-16 h-24 mb-3 preserve-3d transition-transform duration-700" :class="{'rotate-y-180': roomState.revealed && player.hasVoted, 'opacity-40': roomState.revealed && !player.hasVoted}">
                  <!-- Back of card (Face down) -->
                  <div class="absolute inset-0 backface-hidden w-full h-full bg-gradient-to-br from-indigo-500 to-purple-600 rounded-lg shadow-lg border border-white/20 flex items-center justify-center" :class="{ 'opacity-20 border-dashed border-slate-600 bg-none bg-slate-800/50': !player.hasVoted }">
                      <div v-if="player.hasVoted" class="w-10 h-16 border-2 border-white/20 rounded opacity-50 bg-[url('data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSI4IiBoZWlnaHQ9IjgiPgo8cmVjdCB3aWR0aD0iOCIgaGVpZ2h0PSI4IiBmaWxsPSIjZmZmIiBmaWxsLW9wYWNpdHk9IjAuMSIvPgo8L3N2Zz4=')]"></div>
                  </div>
                  <!-- Front of card (Revealed value) -->
                  <div class="absolute inset-0 backface-hidden rotate-y-180 w-full h-full bg-white rounded-lg shadow-xl flex items-center justify-center">
                      <span class="text-3xl font-bold text-indigo-900">{{ getPlayerVote(player.id) }}</span>
                  </div>
               </div>

               <!-- Name -->
               <span class="text-sm font-medium px-3 py-1 rounded-full bg-slate-800/80 border border-slate-700 text-slate-300 truncate max-w-[100px]" :class="{'text-indigo-300 bg-indigo-900/30 border-indigo-500/30': player.id === currentUserId}">
                 {{ player.name }}
               </span>
            </div>
        </div>
      </div>

      <!-- Voting Cards Panel -->
      <div class="bg-white/5 backdrop-blur-md border border-white/10 rounded-3xl p-6 shadow-xl flex flex-col">
        <h3 class="text-lg font-medium text-slate-300 mb-6">Your Estimate</h3>
        
        <div class="grid grid-cols-3 gap-3 mb-6">
           <button v-for="val in deckOptions" :key="val" @click="submitVote(val)" :disabled="roomState.revealed"
            class="relative h-20 bg-slate-800 border border-slate-700 rounded-xl flex items-center justify-center text-2xl font-bold hover:bg-slate-700 hover:-translate-y-1 hover:border-indigo-500/50 transition-all disabled:opacity-50 disabled:transform-none disabled:hover:bg-slate-800 disabled:hover:border-slate-700 shadow-lg"
            :class="{'bg-indigo-600 border-indigo-400 text-white shadow-[0_0_15px_rgba(99,102,241,0.5)] scale-[1.02]': myVote === val}">
              {{ val }}
           </button>
        </div>
        
        <button @click="submitVote(null)" :disabled="roomState.revealed" class="w-full py-3 bg-slate-800 hover:bg-slate-700 border border-slate-700 text-slate-300 rounded-xl transition-all disabled:opacity-50 font-medium">
          Pass
        </button>
      </div>
    </div>

    <!-- Join Modal -->
    <div v-if="!joined" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/80 backdrop-blur-sm transition-opacity">
       <div class="bg-slate-800 border border-slate-700 p-8 rounded-3xl shadow-2xl max-w-md w-full relative overflow-hidden">
         <div class="absolute top-0 right-0 p-12 bg-indigo-500/10 blur-3xl rounded-full"></div>
         <h2 class="text-3xl font-extrabold text-white mb-2 relative">Join Room</h2>
         <p class="text-slate-400 mb-6 font-medium relative">Please enter your name to join the session.</p>
         <form @submit.prevent="joinRoom" class="relative">
           <input v-model="nameInput" autofocus type="text" maxlength="30" placeholder="Your Name" class="w-full bg-slate-900/50 border border-slate-700 rounded-xl px-4 py-3 text-white placeholder-slate-500 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all mb-6">
           <button type="submit" :disabled="!nameInput.trim()" class="w-full py-3 px-4 bg-indigo-600 hover:bg-indigo-500 text-white rounded-xl font-bold shadow-[0_0_20px_-5px_rgba(99,102,241,0.5)] transition-all disabled:opacity-50">
             Join Now
           </button>
         </form>
       </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'

const route = useRoute()
const roomId = route.params.id

const joined = ref(false)
const nameInput = ref('')
const userName = ref('')

// Manager token is a secret UUID, not a simple boolean
const managerToken = ref(localStorage.getItem('managerToken_' + roomId) || '')
const isManager = ref(false) // Will be determined server-side via ROOM_STATE
const currentUserId = ref(null)

const ws = ref(null)
const connectionStatus = ref('idle') // idle | connected | disconnected | error
const errorMessage = ref('')
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5
let reconnectTimer = null

const roomState = ref({
  players: [],
  revealed: false
})
const votesMap = ref({})
const average = ref(0)
const myVote = ref(undefined)

const deckOptions = [1, 2, 3, 5, 8, 13, 21]

onMounted(() => {
  const savedName = localStorage.getItem('username')
  if (savedName) {
    nameInput.value = savedName
  }
})

const getPlayerVote = (id) => {
  const val = votesMap.value[id]
  return val === null ? 'Pass' : val
}

const connectWebSocket = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    ws.value = new WebSocket(`${protocol}//${host}/ws`)

    ws.value.onopen = () => {
        connectionStatus.value = 'connected'
        reconnectAttempts.value = 0

        ws.value.send(JSON.stringify({
            action: 'JOIN_ROOM',
            payload: {
                roomId,
                userName: userName.value,
                managerToken: managerToken.value // Send the secret token, backend verifies it
            }
        }))
    }

    ws.value.onmessage = (event) => {
        const msg = JSON.parse(event.data)
        
        switch (msg.action) {
            case 'JOINED':
                // Backend confirms our user ID
                currentUserId.value = msg.payload.userId
                break
            case 'ROOM_STATE':
                roomState.value = msg.payload
                // Check if we are manager based on the server-assigned flag
                if (currentUserId.value) {
                    const me = msg.payload.players.find(p => p.id === currentUserId.value)
                    if (me) {
                        isManager.value = me.isManager
                    }
                }
                break
            case 'USER_UPDATED':
                const idx = roomState.value.players.findIndex(p => p.id === msg.payload.user.id)
                if (idx !== -1) {
                    roomState.value.players[idx] = msg.payload.user
                } else {
                    roomState.value.players.push(msg.payload.user)
                }
                break
            case 'CARDS_REVEALED':
                roomState.value.revealed = true
                average.value = msg.payload.average
                msg.payload.votes.forEach(v => {
                    votesMap.value[v.userId] = v.voteValue
                })
                break
            case 'ROOM_RESET':
                roomState.value.revealed = false
                votesMap.value = {}
                average.value = 0
                myVote.value = undefined
                roomState.value.players.forEach(p => p.hasVoted = false)
                break
            case 'ERROR':
                errorMessage.value = msg.payload.message
                connectionStatus.value = 'error'
                break
        }
    }

    ws.value.onclose = () => {
        console.log("WebSocket Disconnected")
        if (joined.value && connectionStatus.value !== 'error') {
            connectionStatus.value = 'disconnected'
            attemptReconnect()
        }
    }

    ws.value.onerror = () => {
        console.error("WebSocket Error")
    }
}

const attemptReconnect = () => {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
        connectionStatus.value = 'error'
        errorMessage.value = 'Unable to reconnect. Please refresh the page.'
        return
    }

    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 10000) // Exponential backoff, max 10s
    reconnectAttempts.value++
    console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts.value}/${maxReconnectAttempts})`)

    reconnectTimer = setTimeout(() => {
        connectWebSocket()
    }, delay)
}

const joinRoom = () => {
  if (!nameInput.value.trim()) return
  userName.value = nameInput.value.trim().substring(0, 30) // Client-side limit too
  localStorage.setItem('username', userName.value)
  joined.value = true
  connectWebSocket()
}

const submitVote = (val) => {
    myVote.value = val
    ws.value.send(JSON.stringify({
        action: 'SUBMIT_VOTE',
        payload: {
            voteValue: val
        }
    }))
}

const revealCards = () => {
    ws.value.send(JSON.stringify({ action: 'REVEAL_CARDS', payload: {} }))
}

const resetRoom = () => {
    ws.value.send(JSON.stringify({ action: 'RESET_ROOM', payload: {} }))
}

onUnmounted(() => {
    if (reconnectTimer) clearTimeout(reconnectTimer)
    if (ws.value) ws.value.close()
})

</script>

<style scoped>
@keyframes fadeIn {
  from { opacity: 0; transform: scale(0.9); }
  to { opacity: 1; transform: scale(1); }
}
.animate-fade-in {
  animation: fadeIn 0.4s ease-out forwards;
}
</style>
