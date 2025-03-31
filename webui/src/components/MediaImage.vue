<script setup>
import { ref, onMounted, watch, onUnmounted } from 'vue';
import { fetchMedia, cleanupMedia } from '@/services/media-service.js';

const props = defineProps({
  mediaId: {
    type: String,
    required: false,
    default: null
  },
  alt: {
    type: String,
    default: 'Image'
  },
  className: {
    type: String,
    default: ''
  }
});

const imageUrl = ref('');
const loading = ref(true);
const error = ref(false);

const loadImage = async () => {
  loading.value = true;
  error.value = false;
  
  if (!props.mediaId) {
    error.value = true;
    loading.value = false;
    return;
  }
  
  try {
    const url = await fetchMedia(props.mediaId);
    if (url) {
      imageUrl.value = url;
    } else {
      error.value = true;
    }
  } catch (err) {
    console.error('Error loading image:', err);
    error.value = true;
  } finally {
    loading.value = false;
  }
};

// Watch for changes in mediaId
watch(() => props.mediaId, () => {
  loadImage();
});

onMounted(() => {
  loadImage();
});

onUnmounted(() => {
});
</script>

<template>
  <div class="media-image-container">
    <img 
      v-if="!loading && !error" 
      :src="imageUrl" 
      :alt="alt" 
      :class="className" 
      style="object-fit: cover; width: 100%; height: 100%; object-position: center;"
    />
    <div v-else-if="loading" class="media-loading">
      <div class="loader"></div>
    </div>
    <div v-else class="media-error">
      <img 
        src="https://static.vecteezy.com/system/resources/previews/009/292/244/non_2x/default-avatar-icon-of-social-media-user-vector.jpg" 
        :alt="alt" 
        :class="className" 
        style="object-fit: cover; width: 100%; height: 100%; object-position: center;"
      />
    </div>
  </div>
</template>

<style scoped>
.media-image-container {
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  overflow: hidden;
  position: relative;
}

.media-loading {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 40px;
  width: 100%;
  height: 100%;
}

.loader {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3498db;
  border-radius: 50%;
  width: 20px;
  height: 20px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>