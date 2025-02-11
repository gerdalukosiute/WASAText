<script setup>
import { ref, onMounted, watch } from 'vue';
import axios from 'axios';

const props = defineProps({
  messageId: {
    type: String,
    required: true
  },
  senderId: {
    type: String,
    required: true
  },
  initialStatus: {
    type: String,
    required: true
  },
  isGroupChat: {
    type: Boolean,
    default: false
  },
  participantCount: {
    type: Number,
    required: true
  }
});

const emit = defineEmits(['statusUpdated']);

const currentUserId = ref(localStorage.getItem('userId'));
const status = ref(props.initialStatus);
const updateAttempts = ref(0);

const updateMessageStatus = async (newStatus) => {
  try {
    const response = await axios.put(`http://localhost:8080/messages/${props.messageId}/status`, 
      { status: newStatus },
      {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': currentUserId.value
        }
      }
    );
    
    if (response.data && response.data.status) {
      status.value = response.data.status;
      emit('statusUpdated', { messageId: props.messageId, status: status.value });
      console.log(`Status updated to: ${status.value}`);
    } else {
      console.warn('Unexpected response format:', response.data);
    }
  } catch (err) {
    console.error('Error updating message status:', err);
    if (err.response && err.response.status === 500) {
      console.warn('Server error detected. Retrying update in 5 seconds...');
      setTimeout(() => updateMessageStatus(newStatus), 5000);
    }
  }
};

const getMessageStatusIcon = () => {
  switch (status.value) {
    case 'delivered':
      return '✓'; 
    case 'read':
      return '✓✓'; 
    default:
      return '';
  }
};

const getStatusTitle = () => {
  switch (status.value) {
    case 'delivered':
      return props.isGroupChat ? 'Delivered to all recipients' : 'Delivered';
    case 'read':
      return props.isGroupChat ? 'Read by all recipients' : 'Read';
    default:
      return '';
  }
};

const checkAndUpdateMessageStatus = () => {
  if (updateAttempts.value < 3) {
    updateAttempts.value++;
    if (status.value === 'delivered' && props.senderId !== currentUserId.value) {
      updateMessageStatus('read');
    }
  }
};

onMounted(() => {
  updateAttempts.value = 0;
  checkAndUpdateMessageStatus();
});

watch(() => props.initialStatus, (newStatus) => {
  if (newStatus !== status.value) {
    status.value = newStatus;
    updateAttempts.value = 0;
    checkAndUpdateMessageStatus();
  }
});
</script>

<template>
  <span class="message-status" :title="getStatusTitle()">{{ getMessageStatusIcon() }}</span>
</template>


<style scoped>
.message-status {
  font-size: 0.8em;
  color: #666;
  margin-left: 5px;
  cursor: help;
}
</style>