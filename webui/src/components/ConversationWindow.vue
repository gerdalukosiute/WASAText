<script setup>
import { ref, onMounted, watch, nextTick } from 'vue';
import { format } from 'date-fns';
import Comment from '@/components/Comment.vue';
import ForwardDeleteForm from '@/components/ForwardDeleteForm.vue';
import MessageStatusUpdater from '@/components/MessageStatusUpdater.vue';
import api from '@/services/axios.js'

const props = defineProps({
  conversationId: {
    type: String,
    required: true
  },
  conversationTitle: {
    type: String,
    required: true
  }
});

const emit = defineEmits(['close']);

const messages = ref([]);

const conversationDetails = ref({
  id: '',
  title: '',
  isGroup: false,
  groupPhotoId: '',
  createdAt: '',
  participants: [],
  messages: []
});
const newMessage = ref('');
const loading = ref(true);
const error = ref(null);
const currentUserId = ref('');
const showImageInput = ref(false);
const imageUrl = ref('');
const messageArea = ref(null);
const showReactionBar = ref({});
const showReactionDetails = ref({});
const forwardDeleteForms = ref([]);

const fetchConversationDetails = async () => {
  loading.value = true;
  error.value = null;
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }
    currentUserId.value = userId;

    console.log('Fetching conversation:', props.conversationId);

    const response = await api.get(`/conversations/${props.conversationId}`, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    console.log('Fetched conversation:', response.data);

    if (!response.data || !Array.isArray(response.data.messages)) {
      throw new Error('Invalid response from server');
    }

    conversationDetails.value = {
    id: response.data.conversationId,
    title: response.data.title,
    isGroup: response.data.isGroup,
    groupPhotoId: response.data.groupPhotoId,
    createdAt: response.data.createdAt,
    participants: response.data.participants,
    messages: response.data.messages.map(msg => ({
      id: msg.messageId,
      content: msg.content,
      type: msg.type,
      timestamp: msg.timestamp,
      status: msg.status,
      sender: msg.sender.userId,
      senderName: msg.sender.username,
      comments: msg.comments || []
    }))
    };

    // sorting (may be redundant, check)
    conversationDetails.value.messages.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
    messages.value = conversationDetails.value.messages;

    console.log('Processed conversation details:', conversationDetails.value);
  } catch (err) {
    console.error('Error fetching conversation:', err);
    error.value = 'Failed to load conversation. Please try again.';
  } finally {
    loading.value = false;
  }
};

const updateComments = async (messageId, updatedComments) => {
  const messageIndex = messages.value.findIndex(m => m.id === messageId);
  if (messageIndex !== -1) {
    messages.value[messageIndex].comments = updatedComments;
    messages.value = [...messages.value];
    
    // Fetch latest messages after updating comments
    await fetchConversationDetails();
  }
};

const getReactionShortcut = (comments) => {
  if (!comments || comments.length === 0) return '';
  const reactionCounts = {};
  comments.forEach(comment => {
    if (comment.content.length <= 2) {
      reactionCounts[comment.content] = (reactionCounts[comment.content] || 0) + 1;
    }
  });
  return Object.entries(reactionCounts)
    .map(([emoji, count]) => `${emoji}${count}`)
    .join(' ');
};

const toggleReactionBar = (messageId) => {
  showReactionBar.value = {
    ...showReactionBar.value,
    [messageId]: !showReactionBar.value[messageId]
  };
  // Close reaction details when toggling reaction bar
  showReactionDetails.value = {
    ...showReactionDetails.value,
    [messageId]: false
  };
};

const toggleReactionDetails = (messageId) => {
  showReactionDetails.value = {
    ...showReactionDetails.value,
    [messageId]: !showReactionDetails.value[messageId]
  };
  // Close reaction bar when toggling reaction details
  showReactionBar.value = {
    ...showReactionBar.value,
    [messageId]: false
  };
};

const sendMessage = async (content, type = 'text') => {
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    const response = await api.post(`/conversations/${props.conversationId}/messages`, {
      content,
      type: type
    }, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    const newMessage = {
      ...response.data,
      id: response.data.messageId,
      senderName: response.data.sender.username,
      sender: response.data.sender.userId,
      status: 'sent',
      comments: []
    };

    messages.value.push(newMessage);
    newMessage.value = '';
    imageUrl.value = '';
    showImageInput.value = false;

    await fetchConversationDetails();
    
  } catch (err) {
    console.error('Error sending message:', err);
    error.value = 'Failed to send message. Please try again.';
  }
};

const handleSendMessage = () => {
  if (newMessage.value.trim()) {
    sendMessage(newMessage.value, 'text');
    newMessage.value = ' ';
  }
};

const handleSendImage = () => {
  if (imageUrl.value.trim()) {
    sendMessage(imageUrl.value, 'photo');
  }
};

const formatDate = (dateString) => {
  return format(new Date(dateString), 'MMM d, yyyy HH:mm');
};

const scrollToBottom = () => {
  if (messageArea.value) {
    messageArea.value.scrollTop = messageArea.value.scrollHeight;
  }
};

const handleMessageDeleted = (messageId) => {
  messages.value = messages.value.filter(message => message.id !== messageId);
};

const handleMessageForwarded = (messageId) => {
  console.log('Message forwarded:', messageId);
};

const handleStatusUpdate = ({ messageId, status }) => {
  const messageIndex = messages.value.findIndex(m => m.id === messageId);
  if (messageIndex !== -1) {
    messages.value[messageIndex].status = status;
  }
};

onMounted(() => {
  fetchConversationDetails();
  nextTick(() => {
    scrollToBottom();
  });
});

watch(() => props.conversationId, () => {
  fetchConversationDetails();
  nextTick(() => {
    scrollToBottom();
  });
});

watch(messages, () => {
  nextTick(() => {
    scrollToBottom();
  });
});
</script>

<template>
  <div class="conversation-window">
    <div class="conversation-header">
      <h2 class="text-xl font-bold">{{ conversationDetails.title || conversationTitle }}</h2>
      <button @click="$emit('close')" class="close-button">
        <i class="fa-regular fa-circle-xmark"></i>
      </button>
    </div>
    <div class="message-area" ref="messageArea">
      <div v-if="loading" class="loading-indicator">
        <div class="loader"></div>
      </div>
      <div v-else-if="error" class="error-message">
        {{ error }}
      </div>
      <template v-else-if="messages && messages.length">
        <div v-for="message in messages" :key="message.id" class="message-container" :class="{'sent': message.sender === currentUserId, 'received': message.sender !== currentUserId}">
          <div class="message-header">
            <span class="sender-name">{{ message.senderName }}</span>
            <span v-if="message.icon" class="message-icon">{{ message.icon }}</span>
          </div>
          <div class="message-content-wrapper">
            <ForwardDeleteForm 
              :messageId="message.id"
              @messageDeleted="handleMessageDeleted"
              @messageForwarded="handleMessageForwarded"
              @toggleReactionBar="toggleReactionBar"
              ref="forwardDeleteForms">
            <div class="message-content">
              <template v-if="message.type === 'text'">
                <p>{{ message.content }}</p>
              </template>
              <template v-if="message.type === 'photo'">
                <div class="image-container">
                  <img :src="message.content" alt="Shared image" class="message-image" @error="handleImageError">
                </div>
              </template>
            </div>
          </ForwardDeleteForm>
            <div 
              v-if="getReactionShortcut(message.comments)"
              class="reaction-shortcut" 
              @click.stop="toggleReactionDetails(message.id)"
            >
              {{ getReactionShortcut(message.comments) }}
            </div>
          </div>
          <span class="message-time">{{ formatDate(message.timestamp) }}</span>
          <MessageStatusUpdater 
            :messageId="message.id"
            :senderId="message.senderId"
            :initialStatus="message.status"
            :isGroupChat="conversationDetails.isGroup"
            :participantCount="conversationDetails.participants.length"
            @statusUpdated="handleStatusUpdate"
          />
          <Comment 
            :key="`${message.id}-${message.comments.length}`"
            :messageId="message.id" 
            :comments="message.comments"
            @update-comments="updateComments"
            :showReactionBar="showReactionBar[message.id]"
            :showReactionDetails="showReactionDetails[message.id]"
          />
        </div>
      </template>
      <div v-else class="no-messages">
        No messages in this conversation yet.
      </div>
    </div>
    <div class="input-area">
      <div v-if="showImageInput" class="image-input">
        <input
          v-model="imageUrl"
          type="text"
          placeholder="Enter image URL"
          class="image-url-input"
        />
        <div class="image-input-buttons">
          <button @click="handleSendImage" class="send-image-button">Send Image</button>
          <button @click="showImageInput = false" class="cancel-button">Cancel</button>
        </div>
      </div>
      <form @submit.prevent="handleSendMessage" class="message-form">
        <input
          v-model="newMessage"
          type="text"
          placeholder="Type a message..."
          class="message-input"
        />
        <button type="button" @click="showImageInput = true" class="image-button">
          <i class="fa-regular fa-images"></i>
        </button>
        <button type="submit" class="send-button">
          <i class="fas fa-paper-plane"></i>
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.conversation-window {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 90%;
  max-width: 500px;
  height: 80vh;
  background-color: #fff;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid #e2e8f0;
}

.conversation-header {
  padding: 16px;
  background-color: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
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

.message-area {
  flex-grow: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
}

.message-container {
  margin-bottom: 12px;
  max-width: 70%;
  position: relative;
  min-width: 10%;
  
}

.message-container.sent {
  align-self: flex-end;
}

.message-container.received {
  align-self: flex-start;
}

.message-content {
  padding: 8px 12px;
  border-radius: 16px;
  font-size: 0.95rem;
  line-height: 1.3;
  word-wrap: break-word;
}

.sent .message-content {
  background-color: #4a90e2;
  color: #fff;
}

.received .message-content {
  background-color: #e2e8f0;
  color: #1e293b;
}

.message-time {
  font-size: 0.75rem;
  color: #64748b;
  margin-top: 4px;
  display: block;
}

.image-container {
  max-width: 200px;
  max-height: 200px;
  overflow: hidden;
  border-radius: 12px;
}

.message-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.input-area {
  padding: 16px;
  background-color: #f8fafc;
  border-top: 1px solid #e2e8f0;
}

.message-form {
  display: flex;
  gap: 8px;
}

.message-input {
  flex-grow: 1;
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  font-size: 0.95rem;
}

.image-button, .send-button {
  background-color: #4a90e2;
  color: #fff;
  border: none;
  border-radius: 50%;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 0.2s;
}

.image-button:hover, .send-button:hover {
  background-color: #3183e0;
}

.image-input {
  margin-bottom: 12px;
}

.image-url-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 20px;
  font-size: 0.95rem;
  margin-bottom: 8px;
}

.image-input-buttons {
  display: flex;
  gap: 8px;
}

.send-image-button, .cancel-button {
  flex: 1;
  padding: 8px 12px;
  border: none;
  border-radius: 20px;
  font-size: 0.95rem;
  cursor: pointer;
  transition: background-color 0.2s;
}

.send-image-button {
  background-color: #4a90e2;
  color: #fff;
}

.send-image-button:hover {
  background-color: #3183e0;
}

.cancel-button {
  background-color: #e2e8f0;
  color: #1e293b;
}

.cancel-button:hover {
  background-color: #cbd5e1;
}

.loading-indicator, .error-message {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-size: 1rem;
  color: #64748b;
}

.message-reactions {
  font-size: 0.9em;
  margin-top: 4px;
}

.loader {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3b82f6;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.message-sender-info {
  display: flex;
  align-items: center;
  margin-bottom: 4px;
  padding-left: 4px;
}

.sender-name {
  font-size: 0.8rem;
  color: #64748b;
  font-weight: 500;
}

.reaction-shortcut {
  font-size: 0.8rem;
  margin-top: 4px;
  cursor: pointer;
  display: inline-block;
  background-color: rgba(0, 0, 0, 0.05);
  padding: 2px 6px;
  border-radius: 10px;
}

.reaction-shortcut:hover {
  background-color: rgba(0, 0, 0, 0.1);
}

.reaction-shortcut:empty {
  display: none;
}
</style>