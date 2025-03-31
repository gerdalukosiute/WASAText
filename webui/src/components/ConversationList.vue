<script setup>
import { ref, onMounted } from 'vue';
import { format } from 'date-fns';
import ConversationWindow from '@/components/ConversationWindow.vue';
import SetGroupPhotoForm from '@/components/SetGroupPhotoForm.vue';
import AddUserToGroupForm from '@/components/AddUserToGroupForm.vue';
import SetGroupNameForm from '@/components/SetGroupNameForm.vue'
import LeaveGroupModal from '@/components/LeaveGroupModal.vue';
import MediaImage from '@/components/MediaImage.vue';
import api from '@/services/axios.js'

const conversations = ref([]);
const loading = ref(true);
const error = ref(null);
const currentUserId = ref('');
const selectedConversation = ref(null);
const showDropdown = ref({});
const showSetGroupPhotoForm = ref(false);
const selectedGroupId = ref(null);
const showAddUserToGroupForm = ref(false); 
const isSetGroupNameFormOpen = ref(false); 
const showLeaveGroupConfirmation = ref(false);
const photoUrl = ref('');

const fetchConversations = async () => {
  loading.value = true;
  error.value = null;
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }
    currentUserId.value = userId;

    const response = await api.get('/conversations', {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    if (!response.data || !response.data.conversations) {
      throw new Error('Invalid response format. Expected conversations array.');
    }

    const fetchedConversations = response.data.conversations.map(conv => ({
      ...conv,
      id: conv.conversationId,
      lastMessage: conv.lastMessage || null,
    }));

    // Fetch participants for non-group conversations
    await Promise.all(fetchedConversations.map(fetchConversationDetails));

    // Sort conversations
    fetchedConversations.sort((a, b) => {
      const aTime = getConversationTimestamp(a);
      const bTime = getConversationTimestamp(b);
      return bTime.getTime() - aTime.getTime(); // Sort in descending order (newest first)
    });

    conversations.value = fetchedConversations;
  } catch (err) {
    console.error('Error fetching conversations:', err);
    error.value = 'Failed to load conversations. Please try again.';
  } finally {
    loading.value = false;
  }
};

const fetchConversationDetails = async (conversation) => {
  loading.value = true;
  error.value = null;
  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }
    currentUserId.value = userId;

    console.log('Fetching conversation:', conversation.id);

    const response = await api.get(`/conversations/${conversation.id}`, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    console.log('Fetched conversation:', response.data);

    if (!response.data || !Array.isArray(response.data.messages)) {
      throw new Error('Invalid response from server');
    }

    conversation.participants = response.data.participants || [];
    conversation.title = response.data.title;
    conversation.isGroup = response.data.isGroup;
    conversation.groupPhotoId = response.data.groupPhotoId;
    conversation.createdAt = response.data.createdAt;
    conversation.updatedAt = response.data.updatedAt; // Fallback, remove later

    console.log('Processed conversation details:', conversation.participants, conversation.createdAt);
  } catch (err) {
    console.error('Error fetching conversation:', err);
    error.value = 'Failed to load conversation. Please try again.';
  } finally {
    loading.value = false;
  }
};

const formatDate = (conversation) => {
  const timestamp = getConversationTimestamp(conversation);
  if (!isNaN(timestamp.getTime())) {
    return format(timestamp, 'MMM d, yyyy HH:mm');
  }
  return 'Unknown date';
};

const getMessagePreview = (lastMessage) => {
  console.log(lastMessage)
  if (!lastMessage || !lastMessage.content || lastMessage.timestamp === "0001-01-01T00:00:00Z") {
    return 'No messages yet';
  }
  let prefix = '';
  if (lastMessage.senderId === currentUserId.value) {
    prefix = 'You: ';
  } else if (lastMessage.sender) {
    prefix = `${lastMessage.sender}: `;
  }
  let content = '';
  if (lastMessage.type === 'text') {
    content = lastMessage.content.length > 30 ? lastMessage.content.substring(0, 27) + '...' : lastMessage.content;
  } else if (lastMessage.type === 'photo') {
    content = '📷 Photo';
  } else {
    content = 'New message';
  }
  
  return prefix + content;
};

const getConversationTitle = (conversation) => {
  if (conversation.isGroup) {
    return conversation.title || 'Unnamed Group';
  } else {
    return conversation.title || 'Sumting wong';
  }
};

// Updated function to use the media endpoint
const getProfilePhotoId = (conversation) => {
  if (conversation.isGroup) {
    if (conversation.groupPhotoId) {
      return conversation.groupPhotoId
    }
  } else {
    if (conversation.participants && conversation.participants.length > 0) {
      const otherParticipant = conversation.participants.find(p => p.userId !== currentUserId.value);
      if (otherParticipant && otherParticipant.profilePhotoId) {
        return otherParticipant.profilePhotoId;
      }
    }
  }
  return null;
};

const openConversation = (conversation) => {
  selectedConversation.value = conversation;
};

const closeConversation = () => {
  selectedConversation.value = null;
};

const toggleDropdown = (conversationId) => {
  showDropdown.value = {
    ...showDropdown.value,
    [conversationId]: !showDropdown.value[conversationId]
  };
};

const addNewConversation = (newConversation) => {
  newConversation.lastMessage = null; // Ensure new conversations have no last message
  conversations.value.unshift(newConversation);
};

const getConversationTimestamp = (conversation) => {
  if (conversation.lastMessage.content !== " " &&
      conversation.lastMessage.timestamp !== "0001-01-01T00:00:00Z") {
    return new Date(conversation.lastMessage.timestamp);
  }
  return new Date(conversation.createdAt);
};

const openSetGroupPhotoForm = (groupId) => {
  selectedGroupId.value = groupId;
  showSetGroupPhotoForm.value = true;
  showDropdown.value[groupId] = false;
};

const closeSetGroupPhotoForm = () => {
  showSetGroupPhotoForm.value = false;
  selectedGroupId.value = null;
};

const handleGroupPhotoUpdated = async ({ groupId, newPhotoId }) => {
  console.log('group photo updated:', groupId, newPhotoId);
  const updatedConversation = conversations.value.find(c => c.id === groupId);
  if (updatedConversation) {
    updatedConversation.groupPhotoId = newPhotoId;
    conversations.value = [...conversations.value];
    await fetchConversationDetails(updatedConversation);
    conversations.value = [...conversations.value];
  }
  closeSetGroupPhotoForm();
};

const openAddUserToGroupForm = (groupId) => {
  selectedGroupId.value = groupId;
  showAddUserToGroupForm.value = true;
  showDropdown.value[groupId] = false;
};

const closeAddUserToGroupForm = () => {
  showAddUserToGroupForm.value = false;
  selectedGroupId.value = null;
};

const handleUserAdded = async ({ groupId, username }) => {
  const updatedConversation = conversations.value.find(c => c.id === groupId);
  if (updatedConversation) {
    // Refresh the conversation details to get the updated participant list
    await fetchConversationDetails(updatedConversation);
    // Force a re-render of the conversation list
    conversations.value = [...conversations.value];
  }
  closeAddUserToGroupForm();
};

const openSetGroupNameForm = (groupId) => {
  selectedGroupId.value = groupId;
  isSetGroupNameFormOpen.value = true;
  showDropdown.value[groupId] = false;
};

const closeSetGroupNameForm = () => {
  isSetGroupNameFormOpen.value = false;
  selectedGroupId.value = null;
};

const handleGroupNameUpdated = ({ groupId, newGroupName }) => {
  const updatedConversation = conversations.value.find(c => c.id === groupId);
  if (updatedConversation) {
    updatedConversation.title = newGroupName;
    // Force a re-render of the conversation list
    conversations.value = [...conversations.value];
  }
  closeSetGroupNameForm();
};

const getGroupName = (groupId) => {
  const conversation = conversations.value.find(c => c.id === groupId);
  return conversation ? conversation.title : '';
};

const confirmLeaveGroup = (groupId) => {
  selectedGroupId.value = groupId;
  showLeaveGroupConfirmation.value = true;
  showDropdown.value[groupId] = false;
};

const cancelLeaveGroup = () => {
  showLeaveGroupConfirmation.value = false;
  selectedGroupId.value = null;
};

const leaveGroup = async () => {
  if (!selectedGroupId.value) {
    console.error('No group selected to leave');
    return;
  }

  try {
    const userId = localStorage.getItem('userId');
    if (!userId) {
      throw new Error('User not authenticated');
    }

    const response = await api.delete(`/groups/${selectedGroupId.value}`, {
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': userId
      }
    });

    if (response.data.isGroupDeleted) {
      conversations.value = conversations.value.filter(c => c.id !== selectedGroupId.value);
    } else {
      const updatedConversation = conversations.value.find(c => c.id === selectedGroupId.value);
      if (updatedConversation) {
        await fetchConversationDetails(updatedConversation);
        conversations.value = [...conversations.value];
      }
    }

    showLeaveGroupConfirmation.value = false;
    selectedGroupId.value = null;
  } catch (err) {
    console.error('Error leaving group:', err);
    error.value = 'Failed to leave group. Please try again.';
  }
};

onMounted(() => {
  fetchConversations();
});

defineExpose({ addNewConversation, fetchConversationDetails, fetchConversations });
</script>

<template>
  <div class="conversations-container">
    <h2 class="conversations-title">Conversations</h2>
    <div class="conversation-list-wrapper">
      <div v-if="loading" class="loading">
        <div class="loader"></div>
        <p>Loading conversations...</p>
      </div>
      <div v-else-if="error" class="error">{{ error }}</div>
      <div v-else-if="conversations.length === 0" class="no-conversations">No conversations found.</div>
      <div v-else class="conversation-list">
        <div 
          v-for="conversation in conversations" 
          :key="conversation.id" 
          class="conversation-box"
          @click="openConversation(conversation)"
          :class="{ 'bg-blue-100': selectedConversation && selectedConversation.id === conversation.id }"
        >
            <div class="conversation-photo-container">
              <MediaImage :mediaId="getProfilePhotoId(conversation)" :alt="getConversationTitle(conversation)" className="conversation-photo"/>
            </div>
          <div class="conversation-details">
            <h3 class="conversation-title">{{ getConversationTitle(conversation) }}</h3>
            <p class="conversation-preview">
              {{ getMessagePreview(conversation.lastMessage, currentUserId) }}
            </p>
            <p class="conversation-time">
              {{ formatDate(conversation) }}
            </p>
          </div>
          <div v-if="conversation.isGroup" class="group-actions">
            <button @click.stop="toggleDropdown(conversation.id)" class="group-actions-btn">
              ...
            </button>
            <div v-if="showDropdown[conversation.id]" class="group-dropdown">
              <button @click="openSetGroupPhotoForm(conversation.id)" class="dropdown-item">
                <i class="fa-regular fa-images"></i> Set group photo
              </button>
              <button @click="openSetGroupNameForm(conversation.id)" class="dropdown-item">
              <i class="fas fa-edit"></i> Set group name
              </button>
              <button @click="openAddUserToGroupForm(conversation.id)" class="dropdown-item">
                <i class="fa-solid fa-plus"></i> Add a new member
              </button>
              <button @click="confirmLeaveGroup(conversation.id)" class="dropdown-item text-red-500">
                <i class="fa-regular fa-square-minus"></i> Leave group
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Conversation Window -->
    <div v-if="selectedConversation" class="conversation-modal">
      <div class="modal-overlay" @click="closeConversation"></div>
      <div class="modal-container">
        <ConversationWindow 
          :conversation-id="selectedConversation.id"
          :conversation-title="getConversationTitle(selectedConversation)"
          @close="closeConversation"
        />
      </div>
    </div>
    <SetGroupPhotoForm
      v-if="showSetGroupPhotoForm"
      :is-open="showSetGroupPhotoForm"
      :group-id="selectedGroupId"
      @close="closeSetGroupPhotoForm"
      @photo-updated="handleGroupPhotoUpdated"
    />
    <AddUserToGroupForm
      v-if="showAddUserToGroupForm"
      :is-open="showAddUserToGroupForm"
      :group-id="selectedGroupId"
      @close="closeAddUserToGroupForm"
      @user-added="handleUserAdded"
    />
    <SetGroupNameForm
      v-if="isSetGroupNameFormOpen"
      :is-open="isSetGroupNameFormOpen"
      :group-id="selectedGroupId"
      :current-group-name="getGroupName(selectedGroupId)"
      @close="closeSetGroupNameForm"
      @name-updated="handleGroupNameUpdated"
    />
    <LeaveGroupModal
      :is-open="showLeaveGroupConfirmation"
      title="Leave Group"
      message="Are you sure you want to leave this group?"
      confirm-text="Leave"
      cancel-text="Cancel"
      @confirm="leaveGroup"
      @cancel="cancelLeaveGroup"
    />
  </div>
</template>

<style scoped>
.conversations-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 100%;
  background-color: white;
}

.conversations-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1a202c;
  padding: 1.5rem;
  border-bottom: 1px solid #e2e8f0;
}

.conversation-list-wrapper {
  flex-grow: 1;
  overflow: hidden;
  position: relative;
}

.conversation-list {
  height: 100%;
  overflow-y: auto;
  padding-right: 6px; /* Add some padding for the scrollbar */
}

.conversation-box {
  display: flex;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid #e2e8f0;
  transition: background-color 0.2s;
}

.conversation-box:hover {
  background-color: #f7fafc;
}

.conversation-photo {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
  margin-right: 1rem;
}

.conversation-photo-container {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  margin-right: 15px;
  display: flex;
  justify-content: center;
  align-items: center;
}

/*
.conversation-photo {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
*/

.conversation-details {
  flex: 1;
  min-width: 0;
}

.conversation-title {
  font-size: 1rem;
  font-weight: 600;
  color: #2d3748;
  margin-bottom: 0.25rem;
}

.conversation-preview {
  font-size: 0.875rem;
  color: #718096;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conversation-time {
  font-size: 0.75rem;
  color: #a0aec0;
}

.loading, .error, .no-conversations {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  text-align: center;
  color: #718096;
}

.loader {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #3498db;
  border-radius: 50%;
  width: 24px;
  height: 24px;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.group-actions {
  position: relative;
}

.group-actions-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.25rem;
  color: #718096;
  padding: 0.5rem;
}

.group-dropdown {
  position: absolute;
  right: 0;
  top: 100%;
  background-color: white;
  border: 1px solid #e2e8f0;
  border-radius: 0.5rem;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  z-index: 10;
}

.dropdown-item {
  display: flex;
  align-items: center;
  width: 100%;
  padding: 0.75rem 1rem;
  border: none;
  background: none;
  cursor: pointer;
  transition: background-color 0.2s;
  color: #4a5568;
}

.dropdown-item:hover {
  background-color: #f7fafc;
}

.dropdown-item i {
  margin-right: 0.75rem;
  width: 1rem;
}

.conversation-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-overlay {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
}

.modal-container {
  position: relative;
  width: 90%;
  max-width: 800px;
  max-height: 90vh;
  background-color: white;
  border-radius: 1rem;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
  z-index: 1001;
}

.conversation-list::-webkit-scrollbar {
  width: 6px;
}

.conversation-list::-webkit-scrollbar-track {
  background: #f1f1f1;
}

.conversation-list::-webkit-scrollbar-thumb {
  background: #888;
  border-radius: 3px;
}

.conversation-list::-webkit-scrollbar-thumb:hover {
  background: #555;
}
</style>