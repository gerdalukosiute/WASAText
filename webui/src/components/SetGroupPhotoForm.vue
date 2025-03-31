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

const emit = defineEmits(['close', 'photo-updated']);

const selectedFile = ref(null);
const previewUrl = ref('');
const error = ref('');
const success = ref('');
const isUploading = ref(false);

watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    resetForm();
  }
});

const resetForm = () => {
  selectedFile.value = null;
  previewUrl.value = '';
  error.value = '';
  success.value = '';
  isUploading.value = false;
};

const closeForm = () => {
  // Clean up the preview URL to avoid memory leaks
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value);
    previewUrl.value = '';
  }
  emit('close');
};

const handleFileChange = (event) => {
  const file = event.target.files[0];
  if (!file) return;
  
  // Clean up previous preview if exists
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value);
  }
  
  selectedFile.value = file;
  
  previewUrl.value = URL.createObjectURL(file);
  
  error.value = '';
};

const validateFile = (file) => {
  if (!file) {
    return "Please select a photo to upload";
  }
  
  // Check file type
  if (!file.type.startsWith('image/')) {
    return "Selected file is not an image";
  }
  
  // Check file size (max 5MB)
  const maxSize = 5 * 1024 * 1024; // 5MB in bytes
  if (file.size > maxSize) {
    return "Image is too large (max 5MB)";
  }
  
  return null;
};

const handleSubmit = async () => {
  error.value = '';
  success.value = '';
  isUploading.value = true;
  
  const validationError = validateFile(selectedFile.value);
  if (validationError) {
    error.value = validationError;
    isUploading.value = false;
    return;
  }

  const userId = localStorage.getItem('userId');
  if (!userId) {
    error.value = 'User ID not found. Please log in again.';
    isUploading.value = false;
    return;
  }

  try {
    // Create a FormData object to send the file
    const formData = new FormData();
    formData.append('photo', selectedFile.value);

    // Send the request with FormData
    const response = await api.patch(
      `/groups/${props.groupId}`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
          'X-User-ID': userId
        }
      }
    );

    console.log('Server response:', response);

    if (response.data && response.data.newPhotoId) {
      success.value = 'Group photo updated successfully!';
      
      emit('photo-updated', {
        groupId: props.groupId,
        newPhotoId: response.data.newPhotoId
      });
      
      setTimeout(() => {
        closeForm();
      }, 2000);
    } else {
      throw new Error('Unexpected response from server');
    }
  } catch (err) {
    console.error('Error updating group photo:', err);
    if (err.response) {
      switch (err.response.status) {
        case 400:
          error.value = 'Invalid request. Please check the photo and try again.';
          break;
        case 401:
          error.value = 'Unauthorized. Please check your permissions and try again.';
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
    isUploading.value = false;
  }
};
</script>

<template>
  <div v-if="isOpen" class="set-group-photo-form-overlay">
    <div class="set-group-photo-form">
      <h2>Set Group Photo</h2>
      <form @submit.prevent="handleSubmit">
        <!-- Image preview -->
        <div v-if="previewUrl" class="image-preview">
          <img :src="previewUrl" alt="Preview" class="preview-image" />
        </div>
        
        <!-- File input -->
        <div class="form-group">
          <label for="group-photo-upload" class="file-input-label">
            Select Photo
          </label>
          <input
            id="group-photo-upload"
            type="file"
            accept="image/*"
            @change="handleFileChange"
            class="file-input"
          />
        </div>
        
        <div class="form-actions">
          <button type="submit" :disabled="isUploading || !selectedFile">
            {{ isUploading ? 'Uploading...' : 'Upload Photo' }}
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

.image-preview {
  margin-bottom: 15px;
  text-align: center;
}

.preview-image {
  max-width: 200px;
  max-height: 200px;
  border-radius: 8px;
  object-fit: cover;
}

.form-group {
  margin-bottom: 20px;
}

.file-input-label {
  display: inline-block;
  padding: 8px 16px;
  background-color: #f0f0f0;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.3s;
}

.file-input-label:hover {
  background-color: #e0e0e0;
}

.file-input {
  display: none;
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

button[type="submit"]:disabled {
  background-color: #cccccc;
  cursor: not-allowed;
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