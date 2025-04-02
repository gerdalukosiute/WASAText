<script setup>
import { ref, onMounted } from 'vue';
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

const emit = defineEmits(['close']);

const participants = ref([]);
const loading = ref(false);
const error = ref(null);

const fetchGroupMembers = async () => {
  loading.value = true;
  error.value = null;
  
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }
    
    const response = await api.get(`/conversations/${props.groupId}`, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });
    
    if (!response.data || !Array.isArray(response.data.participants)) {
      throw new Error('Invalid response from server');
    }
    
    participants.value = response.data.participants;
  } catch (err) {
    console.error('Error fetching group members:', err);
    error.value = 'Failed to load group members. Please try again.';
  } finally {
    loading.value = false;
  }
};

const closeForm = () => {
  emit('close');
};

onMounted(() => {
  if (props.isOpen) {
    fetchGroupMembers();
  }
});
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click.stop="closeForm">
    <div class="modal-container" @click.stop>
      <div class="modal-header">
        <h2>Group Members</h2>
        <button @click.stop="closeForm" class="close-button">
          <i class="fa-regular fa-circle-xmark"></i>
        </button>
      </div>
      
      <div class="modal-content">
        <div v-if="loading" class="loading">
          <div class="loader"></div>
          <p>Loading group members...</p>
        </div>
        
        <div v-else-if="error" class="error">
          {{ error }}
        </div>
        
        <div v-else-if="participants.length === 0" class="no-members">
          No members found in this group.
        </div>
        
        <ul v-else class="members-list">
          <li v-for="participant in participants" :key="participant.userId" class="member-item">
            <div class="member-info">
              <span class="member-name">{{ participant.username }}</span>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
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

.modal-container {
  background-color: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  width: 90%;
  max-width: 400px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e2e8f0;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
  color: #2d3748;
}

.close-button {
  background: none;
  border: none;
  font-size: 1.2rem;
  color: #64748b;
  cursor: pointer;
  transition: color 0.2s;
}

.close-button:hover {
  color: #334155;
}

.modal-content {
  padding: 16px;
  overflow-y: auto;
  flex-grow: 1;
}

.loading, .error, .no-members {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  text-align: center;
  color: #64748b;
}

.loader {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3498db;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  animation: spin 1s linear infinite;
  margin-bottom: 12px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.members-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.member-item {
  padding: 12px 8px;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
}

.member-item:last-child {
  border-bottom: none;
}

.member-info {
  margin-left: 12px;
  flex-grow: 1;
}

.member-name {
  font-weight: 500;
  color: #2d3748;
}
</style>