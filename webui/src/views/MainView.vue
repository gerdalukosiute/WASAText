<script setup>
import { ref, onMounted } from 'vue';
import axios from 'axios';

const searchQuery = ref('');
const users = ref([]);
const loading = ref(false);
const error = ref(null);
const searchPerformed = ref(false);
const showDropdown = ref(false);
const showSearchModal = ref(false);
const showUpdateUsernameModal = ref(false);
const showUpdatePhotoModal = ref(false);
const username = ref('');
const newUsername = ref('');
const updateUsernameError = ref('');
const photoUrl = ref('');
const newPhotoUrl = ref('');
const updatePhotoError = ref('');

let debounceTimer;

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value;
};

const openSearchModal = () => {
  showSearchModal.value = true;
  showDropdown.value = false;
};

const closeSearchModal = () => {
  showSearchModal.value = false;
};

const createGroup = () => {
  // will implement group creation here
  console.log('Create group');
  showDropdown.value = false;
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
    const response = await axios.put(`http://localhost:8080/user`, 
      { newName: newUsername.value },
      {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      }
    );

    username.value = newUsername.value;
    localStorage.setItem('username', newUsername.value);
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
    const response = await axios.get(`http://localhost:8080/users`, {
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

const openUpdateUsernameModal = () => {
  showUpdateUsernameModal.value = true;
  showDropdown.value = false;
  newUsername.value = username.value;
  updateUsernameError.value = '';
};

const closeUpdateUsernameModal = () => {
  showUpdateUsernameModal.value = false;
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

const openUpdatePhotoModal = () => {
  showUpdatePhotoModal.value = true;
  showDropdown.value = false;
  newPhotoUrl.value = '';
  updatePhotoError.value = '';
};

const closeUpdatePhotoModal = () => {
  showUpdatePhotoModal.value = false;
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
    const response = await axios.put(`http://localhost:8080/user/${userId}`, 
      { photoUrl: newPhotoUrl.value },
      {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      }
    );

    if (response.status >= 200 && response.status < 300) {
      photoUrl.value = newPhotoUrl.value;
      localStorage.setItem(`userPhotoUrl_${userId}`, newPhotoUrl.value);
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

const fetchUserData = async (userId) => {
  try {
    username.value = localStorage.getItem('username') || '';
    photoUrl.value = localStorage.getItem(`userPhotoUrl_${userId}`) || 'https://hebbkx1anhila5yf.public.blob.vercel-storage.com/pic.jpg-RvO6lH0z7IjCio9xsEjOG5WZnwSqYV.jpeg';
  } catch (error) {
    console.error('Error fetching user data:', error);
  }
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
    <div class="profile-section">
      <div class="profile-photo-container">
        <img :src="photoUrl" alt="Profile Photo" class="profile-photo" @error="photoUrl = 'https://hebbkx1anhila5yf.public.blob.vercel-storage.com/pic.jpg-RvO6lH0z7IjCio9xsEjOG5WZnwSqYV.jpeg'" />
      </div>
    </div>
    <h1 class="welcome-header">Welcome, {{ username }}!</h1>
    <div class="action-container">
      <div class="dropdown">
        <button @click="toggleDropdown" class="action-btn more-actions-btn">More Actions</button>
        <div v-if="showDropdown" class="dropdown-content">
          <a href="#" @click.prevent="openSearchModal">
            <i class="fa-brands fa-searchengin"></i>
            Search for users
          </a>
          <a href="#" @click.prevent="createGroup">
            <i class="fa-solid fa-plus"></i>
            Create group
          </a>
          <a href="#" @click.prevent="openUpdatePhotoModal">
            <i class="fa-regular fa-images"></i>
            Set profile photo
          </a>
          <a href="#" @click.prevent="openUpdateUsernameModal">
            <i class="fa-solid fa-user"></i>
            Update username
          </a>
        </div>
      </div>
    </div>
    <div class="content-container">
      <!-- Placeholder for messages -->
      <div class="messages-placeholder">Messages will appear here</div>
    </div>

    <!-- Search Modal -->
    <div v-if="showSearchModal" class="modal">
      <div class="modal-content">
        <span class="close" @click="closeSearchModal">&times;</span>
        <h2>Search Users</h2>
        <div class="search-container">
          <div class="InputContainer">
            <input
              v-model="searchQuery"
              @input="debounceSearch"
              @keydown.enter.prevent
              placeholder="Search for users"
              id="input"
              class="input"
              name="text"
              type="text"
            />
            <label class="labelforsearch" for="input">
              <svg class="searchIcon" viewBox="0 0 512 512">
                <path
                  d="M416 208c0 45.9-14.9 88.3-40 122.7L502.6 457.4c12.5 12.5 12.5 32.8 0 45.3s-32.8 12.5-45.3 0L330.7 376c-34.4 25.2-76.8 40-122.7 40C93.1 416 0 322.9 0 208S93.1 0 208 0S416 93.1 416 208zM208 352a144 144 0 1 0 0-288 144 144 0 1 0 0 288z"
                ></path>
              </svg>
            </label>
          </div>
          <button @click="getAllUsers" class="action-btn all-users-btn">All Users</button>
        </div>
        <div v-if="loading" class="loading">Loading...</div>
        <div v-else-if="error" class="error">{{ error }}</div>
        <ul v-else-if="users.length > 0" class="user-list">
          <li v-for="user in users" :key="user.id" class="user-item">
            {{ user.name }} (ID: {{ user.id }})
          </li>
        </ul>
        <div v-else-if="searchPerformed" class="no-results">No users found</div>
      </div>
    </div>

    <!-- Update Username Modal -->
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

    <!-- Update Photo Modal -->
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
.main-view {
  width: 100%;
  min-height: 100vh;
  position: relative;
  margin: 0;
  padding: 20px;
}

.profile-section {
  position: fixed;
  left: 20px;
  top: 20px;
}

.profile-photo-container {
  width: 150px;
  height: 150px;
  border-radius: 50%;
  overflow: hidden;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  border: 3px solid white;
}

.profile-photo {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.welcome-header {
  text-align: center;
  margin-bottom: 20px;
  font-size: 24px;
  color: #333;
}

.action-container {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 1000;
}

.dropdown {
  position: relative;
  display: inline-block;
}

.action-btn {
  width: auto;
  height: 2.5em;
  border-radius: 30em;
  font-size: 14px;
  font-family: Arial, Helvetica, sans-serif;
  border: none;
  position: relative;
  overflow: hidden;
  z-index: 1;
  box-shadow: 6px 6px 12px #c5c5c5,
              -6px -6px 12px #ffffff;
  background-color: white;
  cursor: pointer;
  transition: all 0.3s ease;
  padding: 0 15px;
}

.action-btn::before {
  content: '';
  width: 0;
  height: 2.5em;
  border-radius: 30em;
  position: absolute;
  top: 0;
  left: 0;
  background-image: linear-gradient(to right, rgb(110, 183, 235) 0%,rgb(163, 123, 195) 100%);
  transition: .5s ease;
  display: block;
  z-index: -1;
}

.action-btn:hover::before {
  width: 100%;
}

.dropdown-content {
  position: absolute;
  right: 0;
  background-color: #f9f9f9;
  min-width: 160px;
  box-shadow: 0px 8px 16px 0px rgba(0,0,0,0.2);
  z-index: 1;
  border-radius: 5px;
}

.dropdown-content a {
  color: black;
  padding: 12px 16px;
  text-decoration: none;
  display: block;
}

.dropdown-content a:hover {
  background-color: #f1f1f1;
}

.dropdown-content a i {
  margin-right: 10px;
  width: 20px;
  text-align: center;
}

.content-container {
  margin-top: 60px;
  margin-left: 190px; 
}

.messages-placeholder {
  text-align: center;
  color: #888;
  font-style: italic;
}

.modal {
  position: fixed;
  z-index: 1001;
  left: 0;
  top: 0;
  width: 100%;
  height: 100%;
  overflow: auto;
  background-color: rgba(0,0,0,0.4);
}

.modal-content {
  background-color: #fefefe;
  margin: 15% auto;
  padding: 20px;
  border: 1px solid #888;
  width: 80%;
  max-width: 600px;
  border-radius: 5px;
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

.search-container,
.update-username-container,
.update-photo-container {
  display: flex;
  gap: 15px;
  align-items: center;
  width: 100%;
  margin-bottom: 20px;
}

.loading, .error, .no-results {
  text-align: left;
  margin: 10px 0;
}

.error {
  color: red;
}

.user-list {
  list-style-type: none;
  padding: 0;
  margin: 0;
}

.user-item {
  background-color: #f0f0f0;
  margin-bottom: 10px;
  padding: 10px;
  border-radius: 4px;
}

.InputContainer {
  height: 40px;
  display: flex;
  align-items: center;
  background-color: rgb(255, 255, 255);
  border-radius: 10px;
  overflow: hidden;
  cursor: pointer;
  padding-left: 15px;
  box-shadow: 2px 2px 10px rgba(0, 0, 0, 0.075);
  flex-grow: 1;
}

.input {
  width: 100%;
  height: 100%;
  border: none;
  outline: none;
  font-size: 0.9em;
  caret-color: rgb(255, 81, 0);
}

.labelforsearch {
  cursor: text;
  padding: 0px 12px;
}

.searchIcon {
  width: 13px;
}

.searchIcon path {
  fill: rgb(114, 114, 114);
}

.input-wrapper {
  flex-grow: 1;
  position: relative;
}

.styled-input {
  width: 100%;
  padding: 10px 15px;
  border: 2px solid #e0e0e0;
  border-radius: 30px;
  font-size: 16px;
  transition: all 0.3s ease;
  outline: none;
}

.styled-input:focus {
  border-color: #a37bc3;
  box-shadow: 0 0 0 2px rgba(163, 123, 195, 0.2);
}

.update-username-btn,
.update-photo-btn {
  flex-shrink: 0;
}

.action-btn.update-username-btn,
.action-btn.update-photo-btn {
  height: 2.5em;
  border-radius: 30em;
  font-size: 14px;
  font-family: Arial, Helvetica, sans-serif;
  border: none;
  position: relative;
  overflow: hidden;
  z-index: 1;
  box-shadow: 6px 6px 12px #c5c5c5,
              -6px -6px 12px #ffffff;
  background-color: white;
  cursor: pointer;
  transition: all 0.3s ease;
  padding: 0 15px;
}

.action-btn.update-username-btn::before,
.action-btn.update-photo-btn::before {
  content: '';
  width: 0;
  height: 2.5em;
  border-radius: 30em;
  position: absolute;
  top: 0;
  left: 0;
  background-image: linear-gradient(to right, rgb(110, 183, 235) 0%,rgb(163, 123, 195) 100%);
  transition: .5s ease;
  display: block;
  z-index: -1;
}

.action-btn.update-username-btn:hover::before,
.action-btn.update-photo-btn:hover::before {
  width: 100%;
}
</style>

