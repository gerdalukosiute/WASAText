<script setup>
import { ref, computed } from 'vue';
import api from '@/services/axios.js';
import { useRouter } from 'vue-router';

const emit = defineEmits(['close', 'groupCreated']);
const router = useRouter();

const showCreateGroupModal = ref(false);
const groupTitle = ref('');
const newParticipant = ref('');
const participants = ref([]);
const error = ref('');
const isCreating = ref(false);

const isFormValid = computed(() => {
  return groupTitle.value.trim().length > 0;
});

const openCreateGroupModal = () => {
  showCreateGroupModal.value = true;
  resetForm();
};

const closeCreateGroupModal = () => {
  showCreateGroupModal.value = false;
  resetForm();
  emit('close');
};

const resetForm = () => {
  groupTitle.value = '';
  newParticipant.value = '';
  participants.value = [];
  error.value = '';
  isCreating.value = false;
};

const searchAndAddParticipant = async () => {
  const username = newParticipant.value.trim();
  if (!username) return;

  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    const response = await api.get(`/users?q=${username}`, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    if (response.data.users && response.data.users.length > 0) {
      const user = response.data.users[0];
      if (!participants.value.some(p => p.id === user.id)) {
        participants.value.push({ id: user.id, name: user.name });
      }
      newParticipant.value = '';
    } else {
      error.value = 'User not found';
    }
  } catch (err) {
    console.error('Error searching for user:', err);
    error.value = 'Failed to search for user. Please try again.';
  }
};

const removeParticipant = (id) => {
  participants.value = participants.value.filter(p => p.id !== id);
};

const createGroup = async () => {
  if (!isFormValid.value) return;
  
  error.value = '';
  isCreating.value = true;
  
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    // Create the conversation (group)
    const conversationResponse = await api.post('/conversations', {
      title: groupTitle.value,
      isGroup: true,
      participants: [userId]
    }, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    const groupId = conversationResponse.data.id;

    // Add participants to the group
    for (const participant of participants.value) {
      await api.post(`/groups/${groupId}`, {
        username: participant.name
      }, {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      });
    }

    // Emit event to notify parent component about the new group
    emit('groupCreated', {
      id: groupId,
      title: groupTitle.value,
      participants: [{ id: userId, name: localStorage.getItem('username') }, ...participants.value]
    });

    closeCreateGroupModal();
  } catch (err) {
    console.error('Error creating group:', err);
    if (err.response?.status === 401) {
      error.value = 'Your session has expired. Please log in again.';
      localStorage.removeItem('userId');
      router.push('/');
    } else if (err.response?.data) {
      error.value = err.response.data;
    } else {
      error.value = err.message || 'Failed to create group. Please try again.';
    }
  } finally {
    isCreating.value = false;
  }
};
</script>

<template>
  <div>
    <a href="#" @click.prevent="openCreateGroupModal">
      <i class="fa-solid fa-plus"></i>
      Create group
    </a>
    <div v-if="showCreateGroupModal" class="modal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>Create Group</h2>
          <span class="close" @click="closeCreateGroupModal">&times;</span>
        </div>
        <div class="modal-body">
          <form @submit.prevent="createGroup" class="create-group-form">
            <div class="form-group">
              <label for="group-title">Group Title</label>
              <input 
                id="group-title" 
                v-model="groupTitle" 
                type="text"
                class="form-input"
                placeholder="Enter group title"
                required
              />
            </div>
            <div class="form-group">
              <label>Add Participants</label>
              <div class="participants-input-container">
                <input 
                  v-model="newParticipant"
                  type="text"
                  class="form-input"
                  placeholder="Enter username"
                  @keyup.enter.prevent="searchAndAddParticipant"
                />
                <button 
                  type="button"
                  class="add-participant-btn"
                  @click="searchAndAddParticipant"
                >
                  Add
                </button>
              </div>
              <div v-if="participants.length > 0" class="participants-list">
                <div v-for="participant in participants" 
                     :key="participant.id" 
                     class="participant-tag"
                >
                  {{ participant.name }}
                  <button 
                    type="button"
                    class="remove-participant-btn"
                    @click="removeParticipant(participant.id)"
                  >
                    &times;
                  </button>
                </div>
              </div>
            </div>
            <div v-if="error" class="error-message">{{ error }}</div>
            <div class="form-actions">
              <button type="button" class="cancel-btn" @click="closeCreateGroupModal">Cancel</button>
              <button 
                type="submit" 
                class="create-btn" 
                :disabled="!isFormValid || isCreating"
              >
                {{ isCreating ? 'Creating...' : 'Create Group' }}
              </button>
            </div>
          </form>
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

.modal-body {
  padding: 20px;
  overflow-y: auto;
}

.create-group-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-weight: 500;
  color: #333;
}

.form-input {
  width: 100%;
  padding: 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 1rem;
  outline: none;
}

.form-input:focus {
  border-color: #4a90e2;
  box-shadow: 0 0 0 2px rgba(74, 144, 226, 0.2);
}

.participants-input-container {
  display: flex;
  gap: 10px;
}

.add-participant-btn {
  padding: 10px 20px;
  background-color: #4a90e2;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
  white-space: nowrap;
}

.add-participant-btn:hover {
  background-color: #357abd;
}

.participants-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.participant-tag {
  background-color: #e8f0fe;
  color: #1a73e8;
  padding: 4px 8px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 0.9rem;
}

.remove-participant-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 0 4px;
  font-size: 1.2rem;
  line-height: 1;
}

.remove-participant-btn:hover {
  color: #333;
}

.error-message {
  color: #dc3545;
  font-size: 0.9rem;
  margin-top: -10px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 10px;
}

.cancel-btn, .create-btn {
  padding: 10px 20px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.2s;
}

.cancel-btn {
  background-color: #f8f9fa;
  border: 1px solid #ddd;
  color: #333;
}

.cancel-btn:hover {
  background-color: #e9ecef;
}

.create-btn {
  background-color: #4a90e2;
  border: none;
  color: white;
}

.create-btn:hover:not(:disabled) {
  background-color: #357abd;
}

.create-btn:disabled {
  background-color: #a0c4e9;
  cursor: not-allowed;
}
</style>

