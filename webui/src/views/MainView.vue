<script setup>
import { ref, onMounted, onUnmounted } from 'vue';
import ConversationList from '@/components/ConversationList.vue';
import SearchUsersForm from '@/components/SearchUsersForm.vue';
import CreateGroupForm from '@/components/CreateGroupForm.vue';
import SetProfilePhotoForm from '@/components/SetProfilePhotoForm.vue';
import MediaImage from '@/components/MediaImage.vue';
import UpdateUsernameForm from '@/components/UpdateUsernameForm.vue';
import { cleanupAllMedia } from '@/services/media-service.js';
import { useRouter } from 'vue-router';
import api from '@/services/axios.js';

const router = useRouter();

const username = ref('');
const showDropdown = ref(false);
const listKey = ref(0);
const conversations = ref([]);
const userPhotoId = ref('');

const conversationListRef = ref(null);

const fetchUserData = async (userId) => {
  try {
    username.value = localStorage.getItem('username') || '';
    // Get the photo ID from localStorage
    const photoId = localStorage.getItem(`userPhotoId_${userId}`);
    if (photoId) {
      userPhotoId.value = photoId; 
    } else {
      userPhotoId.value = '';
      await fetchUserPhotoFromUsers(userId);
    }
  } catch (error) {
    console.error('Error fetching user data:', error);
  }
};

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value;
};

const closeDropdown = () => {
  showDropdown.value = false;
};

const updateUsername = (newUsername) => {
  username.value = newUsername;
  listKey.value += 1;
};

const updatePhotoUrl = (newPhotoId) => {
 if (newPhotoId) {
   if (typeof newPhotoId === 'string' && (newPhotoId.startsWith('blob:') || newPhotoId.startsWith('/'))) {
     const userId = localStorage.getItem('userId');
     if (userId) {
       if (newPhotoId.startsWith('/media/')) {
         const idMatch = newPhotoId.match(/\/media\/([^?]+)/);
         if (idMatch && idMatch[1]) {
           userPhotoId.value = idMatch[1];
           localStorage.setItem(`userPhotoId_${userId}`, idMatch[1]);
         }
       }
     }
   } else {
     userPhotoId.value = newPhotoId;
     const userId = localStorage.getItem('userId');
     if (userId) {
       localStorage.setItem(`userPhotoId_${userId}`, newPhotoId);
     }
   }
  
   listKey.value += 1;
 }
};

const handleImageError = () => {
  photoUrl.value = 'https://hebbkx1anhila5yf.public.blob.vercel-storage.com/pic.jpg-RvO6lH0z7IjCio9xsEjOG5WZnwSqYV.jpeg';
};

const handleConversationCreated = async (newConversation) => {
  listKey.value += 1;
};

const fetchConversations = async () => {
  listKey.value += 1;
  if (conversationListRef.value && typeof conversationListRef.value.fetchConversations === 'function') {
    await conversationListRef.value.fetchConversations();

    const userId = localStorage.getItem('userId');
    if (userId && !userPhotoId.value) {
      await fetchUserPhotoFromUsers(userId);
    }
  }
};

const logout = () => {
  // localStorage.removeItem('userId');
  // localStorage.removeItem('username');
  // localStorage.removeItem('userPhotoId');
  localStorage.clear();
  router.push('/');
}

const fetchUserPhotoFromUsers = async (userId) => {
  try {
    const currentUsername = localStorage.getItem('username');
    if (!currentUsername) return;
    
    const response = await api.get(`/users?search=${encodeURIComponent(currentUsername)}`, {
      headers: {
        'X-User-ID': userId
      }
    });
    
    if (response.data && response.data.users && Array.isArray(response.data.users)) {
      const currentUser = response.data.users.find(user => user.userId === userId);
      if (currentUser && currentUser.profilePhotoId) {
        userPhotoId.value = currentUser.profilePhotoId;
        localStorage.setItem(`userPhotoId_${userId}`, currentUser.profilePhotoId);
      }
    }
  } catch (error) {
    console.error('Error fetching user photo from users endpoint:', error);
  }
};

onMounted(async () => {
  const userId = localStorage.getItem('userId');
  if (userId) {
    await fetchUserData(userId);
  }
});

onUnmounted(() => {
  cleanupAllMedia();
});
</script>

<template>
  <div class="main-view">
    <div class="fixed-header">
      <div class="profile-section">
        <div class="profile-photo-container">
          <MediaImage v-if="userPhotoId" :mediaId="userPhotoId" alt="Profile Photo" className="profile-photo" />
          <img v-else src="https://static.vecteezy.com/system/resources/previews/009/292/244/non_2x/default-avatar-icon-of-social-media-user-vector.jpg" alt="Profile Photo" class="profile-photo" />
        </div>
      </div>
      <h1 class="welcome-header">Welcome, {{ username }}!</h1>
      <div class="action-container">
        <div class="dropdown">
          <button @click="toggleDropdown" class="action-btn more-actions-btn">
            More Actions
          </button>
          <div v-if="showDropdown" class="dropdown-content">
            <SearchUsersForm @close="closeDropdown" @conversationCreated="handleConversationCreated" @refreshConversations="fetchConversations"/>
            <CreateGroupForm @close="closeDropdown" @refreshConversations="fetchConversations"/>
            <SetProfilePhotoForm @photoUpdated="updatePhotoUrl" @close="closeDropdown" />
            <UpdateUsernameForm 
              :currentUsername="username" 
              @usernameUpdated="updateUsername" 
              @close="closeDropdown"
            />
            <a href="#" @click.prevent="logout">
              <i class="fa-regular fa-rectangle-xmark"></i>
              Logout
            </a>
          </div>
        </div>
      </div>
    </div>
    <div class="content-container">
      <ConversationList :key="listKey" :conversations="conversations" ref="conversationListRef" />
    </div>
  </div>
</template>

<style scoped>
.main-view {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
}

.fixed-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 80px;
  background-color: white;
  display: flex;
  align-items: center;
  padding: 0 40px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  z-index: 10;
}

.profile-section {
  flex-shrink: 0;
  margin-right: 20px;
}

.profile-photo-container {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  overflow: hidden;
  display: flex;
  justify-content: center;
  align-items: center;
  position: relative;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  border: 2px solid white;
}

.profile-photo {
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center;
}

.welcome-header {
  flex-grow: 1;
  font-size: 24px;
  color: #333;
  margin: 0;
  text-align: center;
  padding: 0 20px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.action-container {
  flex-shrink: 0;
  margin-left: 20px;
}

.action-btn {
  background-color: #4a90e2;
  color: white;
  padding: 8px 16px;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: background-color 0.2s;
}

.action-btn:hover {
  background-color:  #4a90e2;
}

.action-btn i {
  font-size: 16px;
}

.dropdown {
  position: relative;
}

.dropdown-content {
  position: absolute;
  right: 0;
  top: 100%;
  background-color: white;
  min-width: 200px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  margin-top: 8px;
  z-index: 20;
}

.content-container {
  margin-top: 80px;
  flex-grow: 1;
  overflow: hidden;
  padding: 20px 40px;
}

:deep(.modal) {
  z-index: 1000;
}

:deep(.modal-content) {
  z-index: 1001;
}

img {
 object-fit: cover;
 width: 100%;
 height: 100%;
 object-position: center;
}
</style>