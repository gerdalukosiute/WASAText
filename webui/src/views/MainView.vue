<script setup>
import { ref, onMounted } from 'vue';
import ConversationList from '@/components/ConversationList.vue';
import SearchUsersForm from '@/components/SearchUsersForm.vue';
import CreateGroupForm from '@/components/CreateGroupForm.vue';
import SetProfilePhotoForm from '@/components/SetProfilePhotoForm.vue';
import UpdateUsernameForm from '@/components/UpdateUsernameForm.vue';

const username = ref('');
const photoUrl = ref('');
const showDropdown = ref(false);
const listKey = ref(0);
const conversations = ref([]);

const conversationListRef = ref(null);

const fetchUserData = async (userId) => {
  try {
    username.value = localStorage.getItem('username') || '';
    photoUrl.value = localStorage.getItem(`userPhotoUrl_${userId}`) || 'https://hebbkx1anhila5yf.public.blob.vercel-storage.com/pic.jpg-RvO6lH0z7IjCio9xsEjOG5WZnwSqYV.jpeg';
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

const updatePhotoUrl = (newPhotoUrl) => {
  photoUrl.value = newPhotoUrl;
};

const handleImageError = () => {
  photoUrl.value = 'https://hebbkx1anhila5yf.public.blob.vercel-storage.com/pic.jpg-RvO6lH0z7IjCio9xsEjOG5WZnwSqYV.jpeg';
};

const handleConversationCreated = async (newConversation) => {
  listKey.value += 1;
};

const fetchConversations = async () => {
  listKey.value += 1;
};

onMounted(async () => {
  const userId = localStorage.getItem('userId');
  if (userId) {
    await fetchUserData(userId);
  }
});
</script>

<template>
  <div class="main-view">
    <div class="fixed-header">
      <div class="profile-section">
        <div class="profile-photo-container">
          <img :src="photoUrl" alt="Profile Photo" class="profile-photo" @error="handleImageError" />
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
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  border: 2px solid white;
}

.profile-photo {
  width: 100%;
  height: 100%;
  object-fit: cover;
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

/* Ensure modal is always on top */
:deep(.modal) {
  z-index: 1000;
}

:deep(.modal-content) {
  z-index: 1001;
}
</style>

