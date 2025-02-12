<script setup>
import { ref } from 'vue';
import api from '@/services/api.js';
import { useRouter } from 'vue-router';

const emit = defineEmits(['close', 'conversationCreated']);

const searchQuery = ref('');
const users = ref([]);
const loading = ref(false);
const error = ref(null);
const searchPerformed = ref(false);
const showSearchModal = ref(false);
const router = useRouter();

let debounceTimer;

const openSearchModal = () => {
  showSearchModal.value = true;
};

const closeSearchModal = () => {
  showSearchModal.value = false;
  emit('close');
};

const debounceSearch = () => {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    performSearch();
  }, 300);
};

const performSearch = async () => {
  if (searchQuery.value.trim() === '') {
    users.value = [];
    searchPerformed.value = false;
    return;
  }

  await fetchUsers({ q: searchQuery.value });
};

const getAllUsers = async () => {
  await fetchUsers();
};

const fetchUsers = async (params = {}) => {
  loading.value = true;
  error.value = null;
  searchPerformed.value = true;

  const userId = localStorage.getItem('userId');
  if (!userId) {
    error.value = 'User not authenticated. Please log in again.';
    loading.value = false;
    return;
  }

  try {
    const response = await api.get(`users`, {
      params: params,
      headers: {
        'Content-Type': 'application/json',
        'X-Requested-With': 'XMLHttpRequest',
        'X-User-ID': userId
      }
    });

    users.value = response.data.users;
  } catch (err) {
    console.error('Error fetching users:', err);
    if (err.response && err.response.status === 401) {
      error.value = 'Authentication failed. Please log in again.';
    } else {
      error.value = 'Failed to fetch users. Please try again.';
    }
  } finally {
    loading.value = false;
  }
};

const startConversation = async (user) => {
  const userId = localStorage.getItem('userId');
  if (!userId) {
    error.value = 'User not authenticated. Please log in again.';
    return;
  }

  try {
    const response = await api.post('conversations', {
      title: user.name,
      isGroup: false,
      participants: [user.id]
    }, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    // Add the new conversation to the list
    const newConversation = {
      conversationId: response.data.id,
      title: user.name,
      isGroup: false,
      createdAt: new Date().toISOString(),
      participants: [{ id: user.id, name: user.name }, { id: userId, name: localStorage.getItem('username') }],
      lastMessage: null
    };

    // Emit an event to add the new conversation to the list
    emit('conversationCreated', newConversation);

    closeSearchModal();
    router.push({ name: 'conversation', params: { id: response.data.id } });
  } catch (err) {
    console.error('Error starting conversation:', err);
    error.value = 'Failed to start conversation. Please try again.';
  }
};
</script>

<template>
  <div>
    <a href="#" @click.prevent="openSearchModal">
      <i class="fa-brands fa-searchengin"></i>
      Search for users
    </a>
    <div v-if="showSearchModal" class="modal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>Search Users</h2>
          <span class="close" @click="closeSearchModal">&times;</span>
        </div>
        <div class="search-container">
          <div class="search-input-container">
            <input
              v-model="searchQuery"
              @input="debounceSearch"
              @keydown.enter.prevent
              placeholder="Search for users"
              class="search-input"
              type="text"
            />
            <button class="search-button">
              <i class="fa-brands fa-searchengin"></i>
            </button>
          </div>
          <button @click="getAllUsers" class="all-users-button">
            All Users
          </button>
        </div>
        <div class="modal-body">
          <div v-if="loading" class="loading">Loading...</div>
          <div v-else-if="error" class="error">{{ error }}</div>
          <div v-else-if="users.length > 0" class="user-list-wrapper">
            <div class="user-list">
              <div v-for="user in users" :key="user.id" class="user-item">
                <span>{{ user.name }} (ID: {{ user.id }})</span>
                <button @click="startConversation(user)" class="start-conversation-btn">
                  <i class="fas fa-plus"></i>
                </button>
              </div>
            </div>
          </div>
          <div v-else-if="searchPerformed" class="no-results">No users found</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal {
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

.modal-content {
  background-color: white;
  width: 90%;
  max-width: 500px;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  display: flex;
  flex-direction: column;
  max-height: 80vh;
}

.modal-header {
  padding: 20px;
  border-bottom: 1px solid #eee;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.5rem;
  color: #333;
}

.close {
  font-size: 1.5rem;
  color: #666;
  cursor: pointer;
  border: none;
  background: none;
  padding: 0;
}

.close:hover {
  color: #333;
}

.search-container {
  padding: 20px;
  display: flex;
  gap: 10px;
  border-bottom: 1px solid #eee;
}

.search-input-container {
  position: relative;
  flex-grow: 1;
}

.search-input {
  width: 100%;
  padding: 10px 40px 10px 15px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 1rem;
  outline: none;
}

.search-input:focus {
  border-color: #4a90e2;
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.2);
}

.search-button {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
}

.all-users-button {
  padding: 10px 20px;
  background-color: #4a90e2;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
  white-space: nowrap;
}

.all-users-button:hover {
  background-color: #357abd;
}

.modal-body {
  padding: 0;
  overflow-y: auto;
  flex-grow: 1;
  min-height: 0;
}

.user-list-wrapper {
  height: 100%;
}

.user-list {
  padding: 0;
}

.user-item {
  padding: 15px 20px;
  border-bottom: 1px solid #eee;
  color: #333;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.user-item:last-child {
  border-bottom: none;
}

.user-item:hover {
  background-color: #f8f9fa;
}

.start-conversation-btn {
  background-color: #4a90e2;
  color: white;
  border: none;
  border-radius: 50%;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 0.2s;
}

.start-conversation-btn:hover {
  background-color: #357abd;
}

.loading, .error, .no-results {
  padding: 20px;
  text-align: center;
  color: #666;
}

.error {
  color: #dc3545;
}

/* Custom scrollbar styles */
.modal-body {
  scrollbar-width: thin;
  scrollbar-color: #ccc transparent;
}

.modal-body::-webkit-scrollbar {
  width: 6px;
}

.modal-body::-webkit-scrollbar-track {
  background: transparent;
}

.modal-body::-webkit-scrollbar-thumb {
  background-color: #ccc;
  border-radius: 3px;
}

.modal-body::-webkit-scrollbar-thumb:hover {
  background-color: #999;
}
</style>