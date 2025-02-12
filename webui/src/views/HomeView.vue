<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/services/axios.js'

const username = ref('')
const error = ref('')
const router = useRouter()

const MIN_USERNAME_LENGTH = 3
const MAX_USERNAME_LENGTH = 16

const usernameLength = computed(() => username.value.trim().length)

const usernameStatus = computed(() => {
  const length = usernameLength.value
  if (length === 0) return { message: '', color: '' }
  if (length < MIN_USERNAME_LENGTH) return { message: 'Too short', color: 'red' }
  if (length > MAX_USERNAME_LENGTH) return { message: 'Too long', color: 'red' }
  return { message: 'Valid length', color: 'green' }
})

function validateUsername(username) {
  const trimmedUsername = username.trim()
  if (trimmedUsername.length < MIN_USERNAME_LENGTH) {
    return `Username must be at least ${MIN_USERNAME_LENGTH} characters long`
  }
  if (trimmedUsername.length > MAX_USERNAME_LENGTH) {
    return `Username cannot be longer than ${MAX_USERNAME_LENGTH} characters`
  }
  return null
}

async function handleSubmit(e) {
  e.preventDefault()
  error.value = ''

  const trimmedUsername = username.value.trim()
  const validationError = validateUsername(trimmedUsername)
  if (validationError) {
    error.value = validationError
    return
  }

  const requestBody = { username: trimmedUsername }
  console.log('Request body:', requestBody)

  try {
    const response = await api.post('/session', requestBody, {
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      }
    })

    console.log('Login successful:', response.data)
    
    if (response.data.identifier) {
      localStorage.setItem('userId', response.data.identifier)
      localStorage.setItem('username', trimmedUsername) // Store the username
      router.push('/main')
    } else {
      throw new Error('User identifier not received from server')
    }
  } catch (err) {
    console.error('Login error:', err)
    error.value = `Failed to login: ${err.response?.data || err.message}`
  }
}
</script>

<template>
  <div class="home">
    <h1 id="homeh1">Welcome to WASAText!</h1>
    <p id="homep1">Enter your username to start messaging</p>
    
    <form @submit="handleSubmit" class="input__container">
      <label class="input__label">Username</label>
      <input 
        v-model="username"
        placeholder="Enter your username" 
        class="input" 
        name="text" 
        type="text"
        :maxlength="MAX_USERNAME_LENGTH"
      >
      <p class="input__description">What do you want to call yourself?</p>
      <div class="username-status">
        <span :style="{ color: usernameStatus.color }">{{ usernameStatus.message }}</span>
        <span>({{ usernameLength }}/{{ MAX_USERNAME_LENGTH }})</span>
      </div>
      <button type="submit" class="submit-button" :disabled="usernameStatus.color === 'red'">Enter Chat</button>
      <p v-if="error" class="error-message">{{ error }}</p>
    </form>
  </div>
</template>

<style scoped>
#homeh1 {
  font-weight: bold;
  text-align: center;
  margin-bottom: 1rem;
}

#homep1 {
  text-align: center;
  margin-bottom: 2rem;
}

/* From Uiverse.io by EddyBel */
.input__container {
  max-width: 200px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  background: rgba(255, 255, 255, 0.3);
  padding: 15px;
  border-radius: 20px;
  position: relative;
  margin: 0 auto;
}

.input__container::before {
  content: "";
  position: absolute;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
  z-index: -1;
  filter: blur(25px);
  border-radius: 20px;
  background-color: #e499ff;
  background-image: radial-gradient(at 47% 69%, hsla(17,62%,65%,1) 0px, transparent 50%),
    radial-gradient(at 9% 32%, hsla(222,75%,60%,1) 0px, transparent 50%);
}

.input__label {
  display: block;
  margin-left: 0.4em;
  color: #000;
  text-transform: uppercase;
  font-size: 0.9em;
  font-weight: bold;
}

.input__description {
  font-size: 0.6em;
  font-weight: bold;
  text-align: center;
  color: rgba(0, 0, 0, 0.5);
}

.input {
  border: none;
  outline: none;
  width: 100%;
  padding: 0.6em;
  padding-left: 0.9em;
  border-radius: 20px;
  background: #fff;
  transition: background 300ms, color 300ms;
}

.input:hover,.input:focus {
  background: rgb(0, 0, 0);
  color: #fff;
}

.username-status {
  display: flex;
  justify-content: space-between;
  font-size: 0.7em;
  margin-top: 0.5rem;
}

.submit-button {
  width: 100%;
  padding: 0.6em;
  margin-top: 10px;
  border: none;
  border-radius: 20px;
  background-color: rgb(163, 123, 195);
  color: white;
  font-family: sans-serif, Arial, Helvetica;
  font-size: 0.9em;
  cursor: pointer;
  transition: background-color 0.3s;
}

.submit-button:hover:not(:disabled) {
  background-color: rgb(110, 183, 235);
}

.submit-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.error-message {
  color: #ff4444;
  font-size: 0.8em;
  text-align: center;
  margin-top: 0.5rem;
  font-weight: bold;
}
</style>