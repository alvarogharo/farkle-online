<script setup>
import { computed } from 'vue';

const props = defineProps({
  value: {
    type: Number,
    required: true,
  },
  isRolling: {
    type: Boolean,
    default: false,
  },
  selected: {
    type: Boolean,
    default: false,
  },
  compact: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  held: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['toggle']);

const pips = computed(() => {
  switch (props.value) {
    case 1:
      return [0, 0, 0, 0, 1, 0, 0, 0, 0];
    case 2:
      return [1, 0, 0, 0, 0, 0, 0, 0, 1];
    case 3:
      return [1, 0, 0, 0, 1, 0, 0, 0, 1];
    case 4:
      return [1, 0, 1, 0, 0, 0, 1, 0, 1];
    case 5:
      return [1, 0, 1, 0, 1, 0, 1, 0, 1];
    case 6:
      return [1, 0, 1, 1, 0, 1, 1, 0, 1];
    default:
      return [0, 0, 0, 0, 0, 0, 0, 0, 0];
  }
});
</script>

<template>
  <button
    type="button"
    class="dice-wrapper"
    :disabled="disabled"
    :class="{
      'dice-wrapper--selected': selected,
      'dice-wrapper--compact': compact,
      'dice-wrapper--disabled': disabled,
      'dice-wrapper--held': held,
    }"
    @click="disabled ? undefined : emit('toggle')"
  >
    <div
      class="dice"
      :class="{
        'dice--rolling': isRolling,
        'dice--compact': compact,
      }"
    >
      <span
        v-for="(pip, index) in pips"
        :key="index"
        class="pip"
        :class="{ 'pip--visible': pip === 1 }"
      />
    </div>
  </button>
</template>

<style scoped>
.dice-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  background: transparent;
  border: none;
  padding: 0;
  cursor: pointer;
  transition: transform 0.08s ease, box-shadow 0.08s ease;
}

.dice-wrapper--compact {
  gap: 0.25rem;
}

.dice-wrapper:hover {
  transform: translateY(-2px);
}

.dice-wrapper--disabled {
  cursor: default;
}

.dice-wrapper--disabled:hover {
  transform: none;
}

.dice-wrapper:disabled {
  opacity: 0.75;
}

.dice-wrapper--held .dice {
  background: radial-gradient(circle at top left, #374151 0, #1f2937 55%, #111827 100%);
  box-shadow:
    0 8px 16px rgba(0, 0, 0, 0.55),
    inset 0 0 0 1px rgba(148, 163, 184, 0.25);
}

.dice-wrapper--held .pip--visible {
  background: radial-gradient(circle at 30% 30%, #ffffff 0, #e5e7eb 50%, #cbd5e1 100%);
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.2),
    0 1px 1px rgba(0, 0, 0, 0.35);
}

.dice-wrapper--selected .dice {
  box-shadow:
    0 0 0 3px rgba(59, 130, 246, 0.9),
    0 12px 24px rgba(15, 23, 42, 0.9),
    inset 0 0 0 1px rgba(148, 163, 184, 0.5);
}

.dice {
  width: 72px;
  height: 72px;
  border-radius: 1rem;
  background: radial-gradient(circle at top left, #ffffff 0, #f4f4f5 40%, #e5e7eb 100%);
  box-shadow:
    0 10px 20px rgba(0, 0, 0, 0.4),
    inset 0 0 0 1px rgba(148, 163, 184, 0.5);
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  grid-template-rows: repeat(3, 1fr);
  padding: 0.5rem;
  gap: 0.15rem;
}

.dice--compact {
  width: 42px;
  height: 42px;
  padding: 0.32rem;
  border-radius: 0.75rem;
  gap: 0.1rem;
}

.dice--rolling {
  animation: shake 0.25s linear infinite;
}

.pip {
  width: 12px;
  height: 12px;
  margin: auto;
  border-radius: 50%;
  background: transparent;
  box-shadow: none;
  transition: background 0.12s ease, box-shadow 0.12s ease;
}

.dice--compact .pip {
  width: 8px;
  height: 8px;
}

.pip--visible {
  background: radial-gradient(circle at 30% 30%, #ffffff 0, #111827 45%, #020617 100%);
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.3),
    0 1px 2px rgba(0, 0, 0, 0.45);
}

.dice--compact .pip--visible {
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.25),
    0 1px 1px rgba(0, 0, 0, 0.35);
}

@keyframes shake {
  0% {
    transform: translate(0, 0) rotate(0deg);
  }
  25% {
    transform: translate(1px, -1px) rotate(-2deg);
  }
  50% {
    transform: translate(-1px, 1px) rotate(2deg);
  }
  75% {
    transform: translate(1px, 1px) rotate(-1deg);
  }
  100% {
    transform: translate(0, 0) rotate(0deg);
  }
}

@media (max-width: 600px) {
  .dice {
    width: 60px;
    height: 60px;
  }
}
</style>

