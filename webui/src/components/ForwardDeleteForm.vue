<script setup>
import { ref, computed } from 'vue';
import api from '@/services/axios.js';
import { onMounted, onUnmounted } from 'vue';

const props = defineProps({
  messageId: {
    type: String,
    required: true
  },
  senderId: {
    type: String,
    required: false,
    default: ''
  },
  conversationId: {
    type: String,
    required: true
  },
});

console.log('ForwardDeleteForm props:', {
  messageId: props.messageId,
  senderId: props.senderId,
  conversationId: props.conversationId
});


const emit = defineEmits(['messageDeleted', 'messageForwarded', 'toggleReactionBar', 'messageReplied']);

const showPopup = ref(false);
const showForwardDialog = ref(false);
const clickTimer = ref(null);
const clickDelay = 300; // milliseconds
const conversations = ref([]);
const selectedConversation = ref('');
const loading = ref(false);
const error = ref(null);
const popupRef = ref(null);
const showReplyDialog = ref(false);
const replyMessage = ref('');
const replyType = ref('text');
const selectedFile = ref(null);
const previewUrl = ref('');

const currentUserId = ref(localStorage.getItem('userId'));
const isCurrentUserSender = computed(() => {
  return props.senderId === currentUserId.value;
})

const handleClick = () => {
  if (clickTimer.value === null) {
    clickTimer.value = setTimeout(() => {
      emit('toggleReactionBar', props.messageId);
      clickTimer.value = null;
    }, clickDelay);
  }
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


    const response = await api.get('/conversations', {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });


    // Check if response.data has a conversations property
    if (response.data && Array.isArray(response.data.conversations)) {
      // Use the conversations array from the response
      conversations.value = response.data.conversations.map(conv => ({
        id: conv.conversationId,
        title: conv.title || (conv.isGroup ? 'Unnamed Group' : 'Direct Message')
      }));
    } else if (Array.isArray(response.data)) {
      // Fallback to the old format for backward compatibility
      conversations.value = response.data.map(conv => ({
        id: conv.conversationId,
        title: conv.title || (conv.isGroup ? 'Unnamed Group' : 'Direct Message')
      }));
    } else {
      throw new Error('Invalid response format. Expected conversations array.');
    }


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

    const response = await api.post(`/messages/${props.messageId}/forward`, {
      originalMessageId: props.messageId,
      targetConversationId: selectedConversation.value
    }, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    console.log('Message forwarded:', response.data);
    
    // Emit the forwarded message data to the parent component
    emit('messageForwarded', {
      messageId: props.messageId,
      forwardedMessage: {
        id: response.data.newMessageId,
        originalMessageId: response.data.originalMessageId,
        targetConversationId: response.data.targetConversationId,
        originalSender: response.data.originalSender,
        forwardedBy: response.data.forwardedBy,
        content: response.data.content,
        type: response.data.type,
        originalTimestamp: response.data.originalTimestamp,
        forwardedTimestamp: response.data.forwardedTimestamp
      }
    });
    
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

    const response = await api.delete(`/messages/${props.messageId}`, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    console.log('Message deleted:', response.data);
    
    emit('messageDeleted', {
      messageId: response.data.messageId,
      user: response.data.user,
      deletedAt: response.data.deletedAt,
      conversationId: response.data.conversationId
    });
    
    showPopup.value = false;
  } catch (error) {
    console.error('Error deleting message:', error);
    if (error.response && error.response.data && error.response.data.error) {
      alert(`Failed to delete message: ${error.response.data.error}`);
    } else {
      alert('Failed to delete message. Please try again.');
    }
  }
};

const handleFileChange = (event) => {
  const file = event.target.files[0];
  if (!file) return;
 
  // Clean up previous preview if exists
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value);
  }
 
  selectedFile.value = file;
  replyType.value = 'photo';
  previewUrl.value = URL.createObjectURL(file);
};

const resetImageUpload = () => {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value);
  }
  selectedFile.value = null;
  previewUrl.value = '';
  replyType.value = 'text';
};

const handleReply = async () => {
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    console.log('Sending reply to conversation:', props.conversationId);
    
    if (replyType.value === 'text') {
      if (!replyMessage.value.trim()) {
        alert('Please enter a message to reply with.');
        return;
      }
     
      console.log('Reply data:', {
        content: replyMessage.value,
        type: 'text',
        parentMessageId: props.messageId
      });

      const response = await api.post(`/conversations/${props.conversationId}/messages`, {
        content: replyMessage.value,
        type: 'text',
        parentMessageId: props.messageId
      }, {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': userId
        }
      });

      console.log('Message replied:', response.data);
     
      emit('messageReplied', response.data);
    } else if (replyType.value === 'photo') {
      if (!selectedFile.value) {
        alert('Please select an image to reply with.');
        return;
      }
     
      const formData = new FormData();
      formData.append('type', 'photo');
      formData.append('photo', selectedFile.value);
      formData.append('parentMessageId', props.messageId);
     
      console.log('Sending photo reply with parent message ID:', props.messageId);
     
      const response = await api.post(
        `/conversations/${props.conversationId}/messages`,
        formData,
        {
          headers: {
            'Content-Type': 'multipart/form-data',
            'X-User-ID': userId
          }
        }
      );
     
      console.log('Photo reply response:', response.data);
     
      emit('messageReplied', response.data);
    }
    
    closeReplyDialog();
  } catch (error) {
    console.error('Error replying to message:', error);
    if (error.response) {
      console.error('Response status:', error.response.status);
      console.error('Response data:', error.response.data);
    }
    if (error.request) {
      console.error('Request:', error.request);
    }
    if (error.config) {
      console.error('Request config:', error.config);
    }
    alert(`Failed to reply to message: ${error.message}`);
  }
};

const closeReplyDialog = () => {
  showReplyDialog.value = false;
  replyMessage.value = '';
  resetImageUpload();
};

const openReplyDialog = () => {
  showPopup.value = false;
  showReplyDialog.value = true;
};

const showDropdown = () => {
  if (event) {
    event.stopPropagation();
  }
  showPopup.value = true;
};

const hideDropdown = () => {
  showPopup.value = false;
}

const handleOutsideClick = (event) => {
  if (showPopup.value && popupRef.value && !popupRef.value.contains(event.target)) {
    showPopup.value = false;
  }
};

onMounted(() => {
  document.addEventListener('mousedown', handleOutsideClick);
  document.addEventListener('touchstart', handleOutsideClick);
});

onUnmounted(() => {
  document.removeEventListener('mousedown', handleOutsideClick);
  document.removeEventListener('touchstart', handleOutsideClick);
});

defineExpose({
  showDropdown,
  hideDropdown
});
</script>

<template>
  <div>
    <div
      @click="handleClick"
    >
      <slot></slot>
    </div>
    <Teleport to="body">
      <div v-if="showPopup" class="modal-overlay" @click.self="hideDropdown">
        <div class="modal-popup" ref="popupRef" @click.stop>
          <div class="modal-header">
            <h3>Message Options</h3>
            <button @click="hideDropdown" class="close-btn">&times;</button>
          </div>
            <button @click="openForwardDialog" class="popup-button forward-button">
              <i class="fa-regular fa-share-from-square"></i> Forward
            </button>
            <button v-if="isCurrentUserSender" @click="handleDelete" class="popup-button delete-button">
              <i class="fa-regular fa-trash-can"></i> Delete
            </button>
            <button @click="openReplyDialog" class="popup-button reply-button">
              <i class="fa-regular fa-square-caret-right"></i> Reply
            </button>
        </div>
      </div>
    </Teleport> 
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
    <!-- Reply Dialog (new) -->
    <div v-if="showReplyDialog" class="modal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>Reply to Message</h2>
          <button @click="closeReplyDialog" class="close">&times;</button>
        </div>
      <div class="modal-body">
        <!-- Image upload section -->
        <div v-if="replyType === 'photo'" class="image-reply-section">
          <div v-if="previewUrl" class="image-preview">
            <img :src="previewUrl" alt="Preview" class="preview-image" />
          </div>
          <div class="form-actions">
            <button @click="replyType = 'text'" class="cancel-btn">Switch to Text</button>
            <button
              @click="handleReply"
              class="create-btn"
              :disabled="!selectedFile"
            >
              Send Reply
            </button>
          </div>
        </div>

        <!-- Text reply section -->
        <div v-else class="text-reply-section">
          <div class="form-group">
            <label for="reply-message">Your Reply</label>
            <textarea
              id="reply-message"
              v-model="replyMessage"
              class="form-input reply-textarea"
              placeholder="Type your reply..."
              rows="4"
            ></textarea>
          </div>
          <div class="form-actions">
            <div class="file-input-container">
              <label for="reply-image-upload" class="file-input-label">
                <i class="fa-regular fa-images"></i> Add Image
              </label>
              <input id="reply-image-upload" type="file" accept="image/*" @change="handleFileChange" class="file-input" />
            </div>
            <button @click="closeReplyDialog" class="cancel-btn">Cancel</button>
            <button
              @click="handleReply"
              class="create-btn"
              :disabled="!replyMessage.trim()"
            >
              Send Reply
            </button>
          </div>
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

.reply-button {
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

.reply-textarea {
  resize: vertical;
  min-height: 80px;
}

.reply-button i {
  margin-right: 8px;
}

.image-reply-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.image-preview {
  margin-bottom: 10px;
  text-align: center;
}

.preview-image {
  max-width: 100%;
  max-height: 200px;
  border-radius: 8px;
  object-fit: cover;
}

.file-input-container {
  margin-right: auto;
}

.file-input-label {
  display: inline-block;
  padding: 8px 16px;
  background-color: #f0f0f0;
  border-radius: 20px;
  cursor: pointer;
  transition: background-color 0.3s;
  text-align: center;
}

.file-input-label:hover {
  background-color: #e0e0e0;
}

.file-input {
  display: none;
}

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
  z-index: 2000;
}

.modal-popup {
  background-color: white;
  border-radius: 8px;
  width: 90%;
  max-width: 300px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  margin: 0;
  font-size: 1.1rem;
  color: #333;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #666;
}

.modal-content {
  padding: 16px;
}

.modal-popup .popup-button {
  display: block;
  width: 100%;
  padding: 12px;
  margin-bottom: 8px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  text-align: left;
  background-color: #f5f5f5;
  transition: background-color 0.2s;
  font-size: 1rem;
}

.modal-popup .popup-button:last-child {
  margin-bottom: 0;
}

.modal-popup .popup-button:hover {
  background-color: #e8e8e8;
}

.modal-popup .forward-button, 
.modal-popup .reply-button {
  color: #474747;
}

.modal-popup .delete-button {
  color: #e53935;
}

.modal-popup .popup-button i {
  margin-right: 10px;
  width: 20px;
  text-align: center;
}
</style>
