<script setup>
import { ref } from 'vue';
import api from '@/services/axios.js';
import { fetchMedia } from '@/services/media-service.js';

const emit = defineEmits(['photoUpdated', 'close']);

const showUpdatePhotoModal = ref(false);
const selectedFile= ref(null);
const previewUrl = ref('');
const isUploading = ref(false);
const updatePhotoError = ref('');

const openUpdatePhotoModal = () => {
  showUpdatePhotoModal.value = true;
  selectedFile.value = null;
  previewUrl.value = '';
  updatePhotoError.value = '';
};

const closeUpdatePhotoModal = () => {
  showUpdatePhotoModal.value = false;
  if (previewUrl.value){
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
  
  // Create a preview URL for the selected image
  previewUrl.value = URL.createObjectURL(file);
  // Clear any previous errors
  updatePhotoError.value = '';
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

const updatePhoto = async () => {
  updatePhotoError.value = '';
  isUploading.value = true;

  const validationError = validateFile(selectedFile.value);
  if (validationError) {
    updatePhotoError.value = validationError;
    isUploading.value = false;
    return;
  }

  const userId = localStorage.getItem('userId');
  if (!userId) {
    updatePhotoError.value = 'User not authenticated. Please log in again.';
    isUploading.value = false;
    return;
  }

  try {
    // Create a FormData object to send the file
    const formData = new FormData();
    formData.append('photo', selectedFile.value);

    console.log('uploading photo for user:', userId);

    const response = await api.put(`/user/${userId}`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
        'X-User-ID': userId
      }
    });

    console.log('uploading photo response:', response.data);

    if (response.status >= 200 && response.status < 300) {
      const newPhotoId = response.data.newPhotoId;
    
      localStorage.setItem(`userPhotoId_${userId}`, newPhotoId);
      console.log('updated photo Id in local storage:', newPhotoId);
    
      emit('photoUpdated', newPhotoId);
    
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
  } finally {
    isUploading.value = false;
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
         <!-- Image preview -->
         <div v-if="previewUrl" class="preview-wrapper">
           <img :src="previewUrl" alt="Preview" class="preview-image" />
         </div>
        
         <!-- File input styled to look like the original input -->
         <div class="input-wrapper">
           <label for="photo-upload" class="file-input-label">
             <span class="file-input-text">{{ selectedFile ? selectedFile.name : 'Select a photo' }}</span>
             <span class="file-input-button">Browse</span>
           </label>
           <input
             id="photo-upload"
             type="file"
             accept="image/*"
             @change="handleFileChange"
             class="file-input"
           />
         </div>
        
         <button
           @click="updatePhoto"
           class="action-btn update-photo-btn"
           :disabled="isUploading || !selectedFile"
         >
           {{ isUploading ? 'Uploading...' : 'Update' }}
         </button>
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
  position: relative;
}

.preview-wrapper {
 margin-bottom: 15px;
 text-align: center;
}

.preview-image {
  max-width: 200px;
  max-height: 200px;
  border-radius: 5px;
  object-fit: cover;
}

.file-input {
  position: absolute;
  width: 0.1px;
  height: 0.1px;
  opacity: 0;
  overflow: hidden;
  z-index: -1;
}

.file-input-label {
  display: flex;
  width: 100%;
  padding: 0;
  border: 1px solid #ccc;
  border-radius: 5px;
  overflow: hidden;
  cursor: pointer;
}

.file-input-text {
  flex-grow: 1;
  padding: 10px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  color: #555;
}

.file-input-button {
  padding: 10px 15px;
  background-color: #f0f0f0;
  border-left: 1px solid #ccc;
  color: #555;
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

.action-btn:disabled {
  background-color: #cccccc;
  cursor: not-allowed;
}

.error {
  color: red;
  margin-top: 10px;
}
</style>