<script setup>
defineProps({
  isOpen: {
    type: Boolean,
    required: true
  },
  title: {
    type: String,
    required: true
  },
  message: {
    type: String,
    required: true
  },
  confirmText: {
    type: String,
    default: 'Confirm'
  },
  cancelText: {
    type: String,
    default: 'Cancel'
  }
});

const emit = defineEmits(['confirm', 'cancel']);

const onConfirm = () => {
  emit('confirm');
};

const onCancel = () => {
  emit('cancel');
};
</script>

<template>
  <div v-if="isOpen" class="confirmation-modal-overlay">
    <div class="confirmation-modal">
      <h2>{{ title }}</h2>
      <p>{{ message }}</p>
      <div class="confirmation-actions">
        <button @click="onConfirm" class="confirm-button">{{ confirmText }}</button>
        <button @click="onCancel" class="cancel-button">{{ cancelText }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.confirmation-modal-overlay {
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

.confirmation-modal {
  background-color: white;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  width: 90%;
  max-width: 400px;
}

h2 {
  margin-bottom: 10px;
  font-size: 1.5rem;
  color: #333;
}

p {
  margin-bottom: 20px;
  color: #666;
}

.confirmation-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

button {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.confirm-button {
  background-color: #4a90e2;
  color: white;
}

.confirm-button:hover {
  background-color: #4a90e2;
}

.cancel-button {
  background-color: #e0e0e0;
  color: #333;
}

.cancel-button:hover {
  background-color: #bdbdbd;
}
</style>