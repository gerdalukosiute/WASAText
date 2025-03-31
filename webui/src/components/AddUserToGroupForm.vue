<script setup>
import { ref, watch } from 'vue';
import api from '@/services/axios.js';

const props = defineProps({
  isOpen: {
    type: Boolean,
    required: true
  },
  groupId: {
    type: String,
    required: true
  }
});

const emit = defineEmits(['close', 'userAdded']);

const username = ref('');
const error = ref('');
const success = ref('');
const isSubmitting = ref(false);

watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    resetForm();
  }
});

const resetForm = () => {
  username.value = '';
  error.value = '';
  success.value = '';
  isSubmitting.value = false;
};

const closeForm = () => {
  emit('close');
};

const handleSubmit = async () => {
  error.value = '';
  success.value = '';
  isSubmitting.value = true;

  const userId = localStorage.getItem('userId');
  if (!userId) {
    error.value = 'User ID not found. Please log in again.';
    isSubmitting.value = false;
    return;
  }

  try {
    const response = await api.post(
      `/groups/${props.groupId}`,
      { usernames: [username.value] },
      {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      }
    );

    if (response.status === 200) {
      success.value = `User ${username.value} added to the group successfully!`;
      emit('userAdded', { 
        groupId: props.groupId,
        username: username.value
      });
      setTimeout(() => {
        closeForm();
      }, 2000);
    } else {
      throw new Error('Unexpected response from server');
    }
  } catch (err) {
    console.error('Error adding user to group:', err);
    if (err.response) {
      switch (err.response.status) {
        case 400:
          error.value = 'Invalid request. Please check the username and try again.';
          break;
        case 401:
          error.value = 'Unauthorized. Your session may have expired. Please log in again.';
          localStorage.removeItem('userId');
          break;
        case 404:
          error.value = 'Group or user not found. Please check the group ID and username.';
          break;
        case 409:
          error.value = 'User is already a member of this group.';
          break;
        default:
          error.value = `An error occurred while adding the user to the group (Status ${err.response.status}). Please try again.`;
      }
    } else if (err.request) {
      error.value = 'No response received from the server. Please check your connection.';
    } else {
      error.value = `Error: ${err.message}`;
    }
  } finally {
    isSubmitting.value = false;
  }
};
</script>

<template>
  <div v-if="isOpen" class="add-user-to-group-form-overlay">
    <div class="add-user-to-group-form">
      <h2>Add New Member to Group</h2>
      <form @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="username">Username:</label>
          <input
            type="text"
            id="username"
            v-model="username"
            required
            placeholder="Enter the username to add"
          />
        </div>
        <div class="form-actions">
          <button type="submit" :disabled="isSubmitting">
            {{ isSubmitting ? 'Adding...' : 'Add Member' }}
          </button>
          <button type="button" @click="closeForm">Cancel</button>
        </div>
      </form>
      <p v-if="error" class="error-message">{{ error }}</p>
      <p v-if="success" class="success-message">{{ success }}</p>
    </div>
  </div>
</template>

<style scoped>
.add-user-to-group-form-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.add-user-to-group-form {
  background-color: white;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  width: 90%;
  max-width: 400px;
}

h2 {
  margin-bottom: 20px;
  font-size: 1.5rem;
  color: #333;
}

.form-group {
  margin-bottom: 20px;
}

label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
}

input {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

button {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

button[type="submit"] {
  background-color: #4a90e2;
  color: white;
}

button[type="submit"]:hover {
  background-color: #4a90e2;
}

button[type="button"] {
  background-color: #f44336;
  color: white;
}

button[type="button"]:hover {
  background-color: #da190b;
}

.error-message {
  color: #f44336;
  margin-top: 10px;
}

.success-message {
  color: #4CAF50;
  margin-top: 10px;
}
</style>