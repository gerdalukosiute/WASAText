<script setup>
  import { ref, watch } from 'vue';
  import axios from 'axios';
  
  const props = defineProps({
    isOpen: {
      type: Boolean,
      required: true
    },
    groupId: {
      type: String,
      default: ''
    }
  });
  
  const emit = defineEmits(['close', 'photoUpdated']);
  
  const groupPhoto = ref('');
  const error = ref('');
  const success = ref('');
  const isSubmitting = ref(false);
  
  watch(() => props.isOpen, (newValue) => {
    if (newValue) {
      resetForm();
    }
  });
  
  const resetForm = () => {
    groupPhoto.value = '';
    error.value = '';
    success.value = '';
    isSubmitting.value = false;
  };
  
  const closeForm = () => {
    emit('close');
  };
  
  const checkGroupMembership = async () => {
    try {
      const userId = localStorage.getItem('userId');
      const response = await axios.get(`http://localhost:8080/conversations/${props.groupId}`, {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      });
      return response.data.participants.some(p => p.id === userId);
    } catch (err) {
      console.error('Error checking group membership:', err);
      return false;
    }
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
  
    const isMember = await checkGroupMembership();
    if (!isMember) {
      error.value = 'You are not a member of this group.';
      isSubmitting.value = false;
      return;
    }
  
    try {
      const response = await axios.patch(
        `http://localhost:8080/groups/${props.groupId}`,
        { groupPhoto: groupPhoto.value },
        {
          headers: {
            'Content-Type': 'application/json',
            'X-User-ID': userId
          }
        }
      );
  
      console.log('Server response:', response);
  
      if (response.data && response.data.newGroupPhoto) {
        success.value = 'Group photo updated successfully!';
        emit('photoUpdated', { 
          groupId: response.data.groupId,
          oldGroupPhoto: response.data.oldGroupPhoto,
          newGroupPhoto: response.data.newGroupPhoto
        });
        setTimeout(() => {
          closeForm();
        }, 2000);
      } else {
        error.value = 'Failed to update group photo. Please try again.';
      }
    } catch (err) {
      console.error('Error updating group photo:', err);
      if (err.response) {
        switch (err.response.status) {
          case 401:
            error.value = 'Unauthorized. Please check your permissions and try again.';
            break;
          case 403:
            error.value = 'Forbidden. You do not have permission to update this group photo.';
            break;
          case 404:
            error.value = 'Group not found. Please check the group ID.';
            break;
          default:
            error.value = `An error occurred while updating the group photo (Status ${err.response.status}). Please try again.`;
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
    <div v-if="isOpen" class="set-group-photo-form-overlay">
      <div class="set-group-photo-form">
        <h2>Set Group Photo</h2>
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="groupPhoto">Group Photo URL:</label>
            <input
              type="text"
              id="groupPhoto"
              v-model="groupPhoto"
              required
              placeholder="Enter the URL of the new group photo"
            />
          </div>
          <div class="form-actions">
            <button type="submit" :disabled="isSubmitting">
              {{ isSubmitting ? 'Updating...' : 'Update Group Photo' }}
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
  .set-group-photo-form-overlay {
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
  
  .set-group-photo-form {
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
  
  