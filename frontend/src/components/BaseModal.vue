<script setup>
const props = defineProps({
  title: {
    type: String,
    default: '',
  },
  onClose: {
    type: Function,
    required: true,
  },
});
</script>

<template>
  <div
    class="base-modal-overlay"
    role="dialog"
    aria-modal="true"
    @click.self="props.onClose()"
  >
    <div class="base-modal">
      <header class="base-modal__header">
        <h2 class="base-modal__title">
          {{ props.title }}
        </h2>
        <button
          type="button"
          class="base-modal__close"
          aria-label="Close"
          @click="props.onClose()"
        >
          ✕
        </button>
      </header>
      <div class="base-modal__body">
        <slot />
      </div>
      <footer
        v-if="$slots.footer"
        class="base-modal__footer"
      >
        <slot name="footer" />
      </footer>
    </div>
  </div>
</template>

<style scoped>
.base-modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1100;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.7);
  backdrop-filter: blur(6px);
}

.base-modal {
  width: 100%;
  max-width: 420px;
  background: linear-gradient(180deg, rgba(31, 41, 55, 0.96) 0%, rgba(15, 23, 42, 0.99) 100%);
  border-radius: 1rem;
  border: 1px solid rgba(148, 163, 184, 0.45);
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.7);
  padding: 1.25rem 1.5rem 1.25rem;
  color: #e5e7eb;
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.base-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.base-modal__title {
  margin: 0;
  font-size: 1.1rem;
  font-weight: 600;
}

.base-modal__close {
  border: none;
  background: transparent;
  color: #9ca3af;
  cursor: pointer;
  font-size: 1rem;
  padding: 0.1rem 0.35rem;
  border-radius: 999px;
}

.base-modal__close:hover {
  background: rgba(148, 163, 184, 0.15);
  color: #e5e7eb;
}

.base-modal__body {
  font-size: 0.95rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.base-modal__footer {
  margin-top: 1rem;
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}
</style>

