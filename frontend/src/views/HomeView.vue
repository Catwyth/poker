<template>
  <div class="flex flex-col items-center max-w-lg w-full">
    <!-- Hero Section -->
    <div class="text-center mb-12">
      <div class="inline-flex items-center justify-center p-3 bg-indigo-500/20 text-indigo-400 rounded-2xl mb-6 shadow-[0_0_30px_rgba(99,102,241,0.3)] ring-1 ring-indigo-500/30">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
        </svg>
      </div>
      <h1 class="text-5xl font-extrabold tracking-tight mb-4 text-transparent bg-clip-text bg-gradient-to-r from-indigo-300 via-white to-purple-300 drop-shadow-sm">
        Planning Poker
      </h1>
      <p class="text-slate-400 text-lg max-w-md mx-auto leading-relaxed">
        Real-time agile estimation with your team. Elegant, fast, and completely free.
      </p>
    </div>

    <!-- Actions -->
    <div class="w-full glass-panel p-8 rounded-3xl border border-white/10 bg-white/5 backdrop-blur-xl shadow-2xl relative overflow-hidden group">
      <!-- Glow effect -->
      <div class="absolute inset-0 bg-gradient-to-br from-indigo-500/10 to-purple-500/10 opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none"></div>

      <button
        @click="createRoom"
        :disabled="isLoading"
        class="relative w-full py-4 px-6 bg-indigo-600 hover:bg-indigo-500 text-white font-bold rounded-2xl shadow-[0_0_40px_-10px_rgba(99,102,241,0.5)] transform hover:-translate-y-1 hover:shadow-[0_0_60px_-15px_rgba(99,102,241,0.7)] transition-all duration-300 disabled:opacity-50 disabled:transform-none flex justify-center items-center gap-3 overflow-hidden"
      >
        <span v-if="isLoading" class="animate-spin h-5 w-5 border-2 border-white/30 border-t-white rounded-full"></span>
        <span v-else>Create New Room</span>
        <svg v-if="!isLoading" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z" clip-rule="evenodd" />
        </svg>
      </button>
      
      <p class="text-center text-sm text-slate-500 mt-6 font-medium">
        No sign up required. Just click to start.
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const isLoading = ref(false)

const createRoom = async () => {
  if (isLoading.value) return
  isLoading.value = true
  
  try {
    const res = await fetch('/api/rooms', {
      method: 'POST'
    })
    const data = await res.json()
    
    // Store the secret manager token for this room (not just a boolean!)
    localStorage.setItem('managerToken_' + data.roomId, data.managerToken)
    
    router.push(`/room/${data.roomId}`)
  } catch (err) {
    console.error("Failed to create room", err)
    alert("Could not connect to the server.")
  } finally {
    isLoading.value = false
  }
}
</script>

<style scoped>
.glass-panel {
  box-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.3);
}
</style>
