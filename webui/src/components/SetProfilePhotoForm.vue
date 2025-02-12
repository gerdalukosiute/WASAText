<script setup>
import { ref } from 'vue';
import api from '@/services/api.js';

const emit = defineEmits(['photoUpdated', 'close']);

const showUpdatePhotoModal = ref(false);
const newPhotoUrl = ref('');
const updatePhotoError = ref('');

const openUpdatePhotoModal = () => {
  showUpdatePhotoModal.value = true;
  newPhotoUrl.value = '';
  updatePhotoError.value = '';
};

const closeUpdatePhotoModal = () => {
  showUpdatePhotoModal.value = false;
  emit('close');
};

const validatePhotoUrl = (url) => {
  if (!url) {
    return "Photo URL cannot be empty";
  }
  try {
    new URL(url);
    return null;
  } catch {
    return "Invalid photo URL";
  }
};

const updatePhoto = async () => {
  updatePhotoError.value = '';
  const validationError = validatePhotoUrl(newPhotoUrl.value);
  if (validationError) {
    updatePhotoError.value = validationError;
    return;
  }

  const userId = localStorage.getItem('userId');
  if (!userId) {
    updatePhotoError.value = 'User not authenticated. Please log in again.';
    return;
  }

  try {
    const response = await api.put(`user/${userId}`, 
      { photoUrl: newPhotoUrl.value },
      {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      }
    );

    if (response.status >= 200 && response.status < 300) {
      localStorage.setItem(`userPhotoUrl_${userId}`, newPhotoUrl.value);
      emit('photoUpdated', newPhotoUrl.value);
      closeUpdatePhotoModal();
    } else {
      throw new Error('Unexpected response from server');
    }
  } catch (err) {
    console.error('Error updating photo:', err);
    if (err.response && err.response.data) {
      updatePhotoError.value = err.response.data.error || 'Failed to update photo. Please try again.';
    } else {
      updatePhotoError.value = 'Failed to update photo. Please try again.';
    }
  }
};
</script>

<template>
  <div>
    <a href="#" @click.prevent="openUpdatePhotoModal">
      <i class="fa-regular fa-images"></i>
      Set profile photo
    </a>
    <div v-if="showUpdatePhotoModal" class="modal">
      <div class="modal-content">
        <span class="close" @click="closeUpdatePhotoModal">&times;</span>
        <h2>Update Profile Photo</h2>
        <div class="update-photo-container">
          <div class="input-wrapper">
            <input
              v-model="newPhotoUrl"
              placeholder="Enter photo URL"
              class="styled-input"
              type="text"
            />
          </div>
          <button @click="updatePhoto" class="action-btn update-photo-btn">Update</button>
        </div>
        <div v-if="updatePhotoError" class="error">{{ updatePhotoError }}</div>
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

.update-photo-container {
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

