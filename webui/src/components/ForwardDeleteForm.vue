<script setup>
import { ref } from 'vue';
import axios from 'axios';

const props = defineProps({
  messageId: {
    type: String,
    required: true
  }
});

const emit = defineEmits(['messageDeleted', 'messageForwarded', 'toggleReactionBar']);

const showPopup = ref(false);
const showForwardDialog = ref(false);
const clickTimer = ref(null);
const clickDelay = 300; // milliseconds
const conversations = ref([]);
const selectedConversation = ref('');
const loading = ref(false);
const error = ref(null);

const handleClick = () => {
  if (clickTimer.value === null) {
    clickTimer.value = setTimeout(() => {
      emit('toggleReactionBar', props.messageId);
      clickTimer.value = null;
    }, clickDelay);
  }
};

const handleDoubleClick = (event) => {
  event.preventDefault();
  clearTimeout(clickTimer.value);
  clickTimer.value = null;
  showPopup.value = !showPopup.value;
};

const openForwardDialog = () => {
  showPopup.value = false;
  showForwardDialog.value = true;
  fetchConversations();
};

const closeForwardDialog = () => {
  showForwardDialog.value = false;
  selectedConversation.value = '';
};

const fetchConversations = async () => {
  loading.value = true;
  error.value = null;
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    const response = await axios.get('http://localhost:8080/conversations', {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    if (!Array.isArray(response.data)) {
      throw new Error('Invalid response format. Expected an array.');
    }

    conversations.value = response.data.map(conv => ({
      id: conv.conversationId,
      title: conv.title || (conv.isGroup ? 'Unnamed Group' : 'Direct Message')
    }));

  } catch (err) {
    console.error('Error fetching conversations:', err);
    error.value = 'Failed to load conversations. Please try again.';
  } finally {
    loading.value = false;
  }
};

const handleForward = async () => {
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    if (!selectedConversation.value) {
      alert('Please select a conversation to forward the message to.');
      return;
    }

    const response = await axios.post(`http://localhost:8080/messages/${props.messageId}/forward`, {
      originalMessageId: props.messageId,
      targetConversationId: selectedConversation.value
    }, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    console.log('Message forwarded:', response.data);
    emit('messageForwarded', props.messageId);
    closeForwardDialog();
  } catch (error) {
    console.error('Error forwarding message:', error);
    if (error.response && error.response.data && error.response.data.error) {
      alert(`Failed to forward message: ${error.response.data.error}`);
    } else {
      alert('Failed to forward message. Please try again.');
    }
  }
};

const handleDelete = async () => {
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    const response = await axios.delete(`http://localhost:8080/messages/${props.messageId}`, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    console.log('Message deleted:', response.data);
    emit('messageDeleted', props.messageId);
    showPopup.value = false;
  } catch (error) {
    console.error('Error deleting message:', error);
  }
};
</script>

<template>
  <div>
    <div 
      @click="handleClick"
      @dblclick="handleDoubleClick"
    >
      <slot></slot>
    </div>
    <div v-if="showPopup" class="popup">
      <button @click="openForwardDialog" class="popup-button forward-button">
        <i class="fa-regular fa-share-from-square"></i> Forward
      </button>
      <button @click="handleDelete" class="popup-button delete-button">
        <i class="fa-regular fa-trash-can"></i> Delete
      </button>
    </div>
    <div v-if="showForwardDialog" class="modal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>Forward Message</h2>
          <button @click="closeForwardDialog" class="close">&times;</button>
        </div>
        <div class="modal-body">
          <div v-if="loading" class="loading">Loading conversations...</div>
          <div v-else-if="error" class="error-message">{{ error }}</div>
          <div v-else class="form-group">
            <label for="conversation-select">Select a conversation</label>
            <select 
              id="conversation-select"
              v-model="selectedConversation" 
              class="form-input"
            >
              <option value="">Choose a conversation</option>
              <option v-for="conv in conversations" :key="conv.id" :value="conv.id">
                {{ conv.title || 'Unnamed Conversation' }}
              </option>
            </select>
          </div>
          <div class="form-actions">
            <button @click="closeForwardDialog" class="cancel-btn">Cancel</button>
            <button 
              @click="handleForward" 
              class="create-btn" 
              :disabled="!selectedConversation"
            >
              Forward
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.popup {
  position: absolute;
  background-color: white;
  border: 1px solid #ccc;
  border-radius: 5px;
  padding: 10px;
  z-index: 1000;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.popup-button {
  display: block;
  width: 100%;
  padding: 8px 12px;
  margin-bottom: 5px;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  text-align: left;
  background-color: transparent;
  transition: background-color 0.2s;
}

.popup-button:hover {
  background-color: #f0f0f0;
}

.forward-button {
  color: #474747;
}

.delete-button {
  color: #474747;
}

.fas {
  margin-right: 8px;
}

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

.error-message {
  color: #dc3545;
  font-size: 0.9rem;
  margin-top: 10px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
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

.loading {
  text-align: center;
  color: #666;
  font-style: italic;
}
</style>