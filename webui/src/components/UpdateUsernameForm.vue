<script setup>
import { ref } from 'vue';
import api from '@/services/axios.js';

const props = defineProps({
  currentUsername: {
    type: String,
    required: true
  }
});

const emit = defineEmits(['usernameUpdated', 'close']);

const showUpdateUsernameModal = ref(false);
const newUsername = ref('');
const updateUsernameError = ref('');

const openUpdateUsernameModal = () => {
  showUpdateUsernameModal.value = true;
  newUsername.value = props.currentUsername;
  updateUsernameError.value = '';
};

const closeUpdateUsernameModal = () => {
  showUpdateUsernameModal.value = false;
  emit('close');
};

const validateUsername = (username) => {
  if (username.length < 3 || username.length > 16) {
    return "Username must be between 3 and 16 characters";
  }
  if (!/^[a-zA-Z0-9_]+$/.test(username)) {
    return "Username must contain only letters, numbers, and underscores";
  }
  return null;
};

const updateUsername = async () => {
  updateUsernameError.value = '';
  const validationError = validateUsername(newUsername.value);
  if (validationError) {
    updateUsernameError.value = validationError;
    return;
  }

  const userId = localStorage.getItem('userId');
  if (!userId) {
    updateUsernameError.value = 'User not authenticated. Please log in again.';
    return;
  }

  try {
    const response = await api.put(`/user`, 
      { newName: newUsername.value },
      {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      }
    );
    console.log(response)
    localStorage.setItem('username', newUsername.value);
    emit('usernameUpdated', newUsername.value);
    closeUpdateUsernameModal();
  } catch (err) {
    console.error('Error updating username:', err);
    if (err.response) {
      updateUsernameError.value = err.response.data || 'Failed to update username. Please try again.';
    } else {
      updateUsernameError.value = 'Failed to update username. Please try again.';
    }
  }
};
</script>

<template>
  <div>
    <a href="#" @click.prevent="openUpdateUsernameModal">
      <i class="fa-solid fa-user"></i>
      Update username
    </a>
    <div v-if="showUpdateUsernameModal" class="modal">
      <div class="modal-content">
        <span class="close" @click="closeUpdateUsernameModal">&times;</span>
        <h2>Update Username</h2>
        <div class="update-username-container">
          <div class="input-wrapper">
            <input
              v-model="newUsername"
              placeholder="Enter new username"
              class="styled-input"
              type="text"
            />
          </div>
          <button @click="updateUsername" class="action-btn update-username-btn">Update</button>
        </div>
        <div v-if="updateUsernameError" class="error">{{ updateUsernameError }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal {
  display: flex;
  position: fixed;
  z-index: 1000;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  overflow: auto;
  background-color: rgba(0,0,0,0.4);
  align-items: center;
  justify-content: center;
}

.modal-content {
  background-color: #fefefe;
  padding: 20px;
  border: 1px solid #888;
  width: 80%;
  max-width: 500px;
  border-radius: 10px;
}

.close {
  color: #aaa;
  float: right;
  font-size: 28px;
  font-weight: bold;
  cursor: pointer;
}

.close:hover,
.close:focus {
  color: black;
  text-decoration: none;
  cursor: pointer;
}

.update-username-container {
  margin-top: 20px;
}

.input-wrapper {
  margin-bottom: 15px;
}

.styled-input {
  width: 100%;
  padding: 10px;
  border: 1px solid #ccc;
  border-radius: 5px;
  font-size: 16px;
}

.action-btn {
  background-color: #4a90e2;
  color: white;
  padding: 10px 15px;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  font-size: 16px;
}

.error {
  color: red;
  margin-top: 10px;
}
</style>