<script setup>
import { ref, computed } from 'vue';
import api from '@/services/axios.js';

const props = defineProps({
  messageId: {
    type: String,
    required: true
  },
  comments: {
    type: Array,
    default: () => []
  },
  showReactionBar: {
    type: Boolean,
    default: false
  },
  showReactionDetails: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['update-comments']);

const currentUserId = ref(localStorage.getItem('userId'));

const userReaction = computed(() => {
  if (!props.comments) return null;
  const userComment = props.comments.find(c => c.userId === currentUserId.value && c.content.length <= 2);
  return userComment ? userComment.content : null;
});

const reactionCounts = computed(() => {
  if (!props.comments) return {};
  const counts = {};
  props.comments.forEach(comment => {
    if (comment.content.length <= 2) {
      counts[comment.content] = (counts[comment.content] || 0) + 1;
    }
  });
  return counts;
});

const uniqueReactions = computed(() => {
  if (!props.comments) return [];
  const reactions = new Map();
  props.comments.forEach(comment => {
    if (comment.content.length <= 2 && !reactions.has(comment.content)) {
      reactions.set(comment.content, comment);
    }
  });
  return Array.from(reactions.values());
});

const getUsersForReaction = (emoji) => {
  return props.comments.filter(comment => comment.content === emoji);
};

const handleEmojiClick = async (emoji) => {
  try {
    if (userReaction.value === emoji) {
      // Delete the existing reaction
      const commentToDelete = props.comments.find(c => c.userId === currentUserId.value && c.content === emoji);
      if (commentToDelete) {
        await api.delete(`/messages/${props.messageId}/comments/${commentToDelete.id}`, {
          headers: {
            'Content-Type': 'application/json',
            'X-User-ID': currentUserId.value
          }
        });
        const updatedComments = props.comments.filter(c => c.id !== commentToDelete.id);
        emit('update-comments', props.messageId, updatedComments);
      }
    } else {
      // Add new reaction
      const response = await api.post(`/messages/${props.messageId}/comments`, {
        content: emoji
      }, {
        headers: {
          'Content-Type': 'application/json',
          'X-User-ID': currentUserId.value
        }
      });
      const newComment = response.data;
      let updatedComments = props.comments ? [...props.comments] : [];
      if (userReaction.value) {
        // Remove existing reaction
        updatedComments = updatedComments.filter(comment => 
          !(comment.userId === currentUserId.value && comment.content.length <= 2)
        );
      }
      updatedComments.push(newComment);
      emit('update-comments', props.messageId, updatedComments);
    }
  } catch (error) {
    console.error('Error handling emoji reaction:', error);
    if (error.response && error.response.data) {
      console.error('Server error message:', error.response.data);
    }
  }
};
</script>

<template>
  <div class="comment">
    <div v-if="showReactionBar" class="emoji-reactions">
      <button 
        v-for="emoji in ['👍', '❤️', '😂', '😮', '😢', '😡']" 
        :key="emoji" 
        @click="handleEmojiClick(emoji)"
        :class="{ 'selected': userReaction === emoji }"
      >
        {{ emoji }}
        <span v-if="reactionCounts[emoji]" class="reaction-count">{{ reactionCounts[emoji] }}</span>
      </button>
    </div>
    <div v-if="showReactionDetails" class="reaction-details">
      <div v-for="reaction in uniqueReactions" :key="reaction.id" class="reaction-detail">
        {{ reaction.content }} 
        <span v-for="user in getUsersForReaction(reaction.content)" :key="user.id" class="reaction-user">
          {{ user.username }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.comment {
  margin-top: 8px;
}

.emoji-reactions {
  display: flex;
  gap: 4px;
  margin-bottom: 4px;
}

.emoji-reactions button {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 2px 4px;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.emoji-reactions button:hover {
  background-color: rgba(0, 0, 0, 0.1);
}

.emoji-reactions button.selected {
  background-color: rgba(59, 130, 246, 0.2);
}

.reaction-count {
  font-size: 0.8rem;
  margin-left: 2px;
}

.reaction-details {
  font-size: 0.8rem;
  color: #64748b;
  margin-top: 4px;
}

.reaction-detail {
  display: flex;
  align-items: center;
  margin-bottom: 2px;
}

.reaction-user {
  margin-left: 4px;
  background-color: rgba(0, 0, 0, 0.05);
  padding: 1px 4px;
  border-radius: 4px;
}

</style>