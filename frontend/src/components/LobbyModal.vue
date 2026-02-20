<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';

const props = defineProps({
  /** Función para enviar mensajes por WebSocket */
  send: { type: Function, required: true },
  /** Función para registrar handler de mensajes */
  onMessage: { type: Function, required: true },
  /** Estado de conexión */
  connected: { type: Boolean, default: false },
  /** Mensaje de error de conexión */
  lastError: { type: [String, Object], default: null },
});

const emit = defineEmits(['success']);

const tab = ref('create'); // 'create' | 'join'
const waitingForPlayer = ref(false);
const pendingGameCode = ref('');
const joinViaLink = ref(false); // true cuando llegas con ?code=XXX

// Crear partida
const createName = ref('');
const createVictoryScore = ref(2000);
const createLoading = ref(false);

// Unirse
const joinName = ref('');
const joinCode = ref('');
const joinLoading = ref(false);

const serverError = ref('');
let unsubscribe = () => {};

function clearErrors() {
  serverError.value = '';
}

function handleMessage(data) {
  if (data.type === 'error') {
    serverError.value = data.message || 'Error del servidor';
    createLoading.value = false;
    joinLoading.value = false;
    return;
  }
  if (data.type === 'game_created') {
    createLoading.value = false;
    serverError.value = '';
    pendingGameCode.value = data.gameCode || '';
    waitingForPlayer.value = true;
    return;
  }
  if (data.type === 'player_joined') {
    // El creador recibe esto cuando alguien se une; cerramos el modal
    emit('success', { gameCode: pendingGameCode.value, playerIndex: 0 });
    return;
  }
  if (data.type === 'game_joined') {
    joinLoading.value = false;
    emit('success', {
      gameCode: data.gameCode,
      playerIndex: data.playerIndex ?? 1,
    });
  }
}

function capitalizeName(s) {
  return s
    .split(/\s+/)
    .map((w) => (w ? w[0].toUpperCase() + w.slice(1).toLowerCase() : ''))
    .join(' ')
    .trim();
}

function doCreate() {
  clearErrors();
  const name = capitalizeName(createName.value.trim()) || 'Jugador 1';
  const victoryScore = Math.max(100, Math.min(100000, Number(createVictoryScore.value) || 2000));
  createLoading.value = true;
  props.send({ type: 'create', playerName: name, victoryScore });
}

function doJoin() {
  clearErrors();
  const name = capitalizeName(joinName.value.trim()) || 'Jugador 2';
  const code = joinCode.value.trim().toUpperCase();
  if (!code) {
    serverError.value = 'Introduce el código de la partida';
    return;
  }
  joinLoading.value = true;
  props.send({ type: 'join', gameCode: code, playerName: name });
}

const shareUrl = computed(() => {
  if (!pendingGameCode.value) return '';
  const base = typeof window !== 'undefined'
    ? `${window.location.origin}${window.location.pathname}`
    : '';
  return `${base}?code=${encodeURIComponent(pendingGameCode.value)}`;
});

async function copyLink() {
  if (!shareUrl.value) return;
  try {
    await navigator.clipboard.writeText(shareUrl.value);
    linkCopyFeedback.value = true;
    setTimeout(() => { linkCopyFeedback.value = false; }, 1500);
  } catch {
    copyCode(); // fallback: copiar solo el código
  }
}

const linkCopyFeedback = ref(false);

onMounted(() => {
  unsubscribe = props.onMessage(handleMessage);
  const params = new URLSearchParams(window.location.search);
  const codeFromUrl = params.get('code')?.trim().toUpperCase();
  if (codeFromUrl) {
    tab.value = 'join';
    joinCode.value = codeFromUrl;
    joinViaLink.value = true;
  }
});

async function copyCode() {
  if (!pendingGameCode.value) return;
  try {
    await navigator.clipboard.writeText(pendingGameCode.value);
    copyFeedback.value = true;
    setTimeout(() => { copyFeedback.value = false; }, 1500);
  } catch {
    // fallback: seleccionar el texto
    const el = document.querySelector('.code-display');
    if (el) {
      const range = document.createRange();
      range.selectNodeContents(el);
      window.getSelection()?.removeAllRanges();
      window.getSelection()?.addRange(range);
    }
  }
}

const copyFeedback = ref(false);

onUnmounted(() => {
  unsubscribe();
});
</script>

<template>
  <div class="lobby-overlay" role="dialog" aria-modal="true" aria-labelledby="lobby-title">
    <div class="lobby-modal">
      <h1 id="lobby-title" class="lobby-title">Farkle</h1>
      <p class="lobby-subtitle">Crea una partida o únete a una existente</p>

      <div class="connection-status" :class="{ 'connection-status--ok': connected, 'connection-status--error': lastError }">
        <span v-if="connected">Conectado</span>
        <span v-else-if="lastError">Error de conexión</span>
        <span v-else>Conectando…</span>
      </div>

      <div v-if="serverError" class="lobby-error">
        {{ serverError }}
      </div>

      <!-- Pantalla de espera tras crear partida -->
      <div v-if="waitingForPlayer" class="waiting-section">
        <p class="waiting-text">Comparte este código o enlace para que alguien se una a la partida:</p>
        <div class="code-display" :class="{ 'code-display--copied': copyFeedback }">
          {{ pendingGameCode }}
        </div>
        <div class="waiting-buttons">
          <button
            type="button"
            class="btn btn--copy"
            :class="{ 'btn--copied': copyFeedback }"
            @click="copyCode"
          >
            {{ copyFeedback ? '¡Copiado!' : 'Copiar código' }}
          </button>
          <button
            type="button"
            class="btn btn--copy"
            :class="{ 'btn--copied': linkCopyFeedback }"
            @click="copyLink"
          >
            {{ linkCopyFeedback ? '¡Enlace copiado!' : 'Copiar enlace' }}
          </button>
        </div>
        <p class="waiting-hint">Esperando que alguien se una…</p>
      </div>

      <template v-else>
      <div class="lobby-tabs">
        <button
          type="button"
          class="tab-btn"
          :class="{ 'tab-btn--active': tab === 'create' }"
          @click="tab = 'create'; clearErrors()"
        >
          Crear partida
        </button>
        <button
          type="button"
          class="tab-btn"
          :class="{ 'tab-btn--active': tab === 'join' }"
          @click="tab = 'join'; clearErrors()"
        >
          Unirse a partida
        </button>
      </div>

      <form v-if="tab === 'create'" class="lobby-form" @submit.prevent="doCreate">
        <label class="lobby-label">
          Nombre
          <input
            v-model="createName"
            type="text"
            class="lobby-input"
            placeholder="Jugador 1"
            :disabled="!connected"
          >
        </label>
        <label class="lobby-label">
          Puntuación para ganar
          <input
            v-model.number="createVictoryScore"
            type="number"
            class="lobby-input"
            min="100"
            max="100000"
            :disabled="!connected"
          >
        </label>
        <button
          type="submit"
          class="btn btn--primary"
          :disabled="!connected || createLoading"
        >
          {{ createLoading ? 'Creando…' : 'Crear partida' }}
        </button>
      </form>

      <form v-else class="lobby-form" @submit.prevent="doJoin">
        <p v-if="joinViaLink" class="join-link-intro">
          Has llegado mediante un enlace. Introduce tu nombre para unirte a la partida.
        </p>
        <label class="lobby-label">
          Nombre
          <input
            v-model="joinName"
            type="text"
            class="lobby-input"
            :placeholder="joinViaLink ? 'Tu nombre' : 'Jugador 2'"
            :disabled="!connected"
          >
        </label>
        <label v-if="!joinViaLink" class="lobby-label">
          Código de partida
          <input
            v-model="joinCode"
            type="text"
            class="lobby-input lobby-input--code"
            placeholder="ABCDE"
            maxlength="5"
            :disabled="!connected"
            @input="joinCode = joinCode.toUpperCase()"
          >
        </label>
        <button
          type="submit"
          class="btn btn--primary"
          :disabled="!connected || joinLoading || (!joinViaLink && !joinCode.trim())"
        >
          {{ joinLoading ? 'Uniéndose…' : 'Unirse' }}
        </button>
      </form>
      </template>
    </div>
  </div>
</template>

<style scoped>
.lobby-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(2, 4, 11, 0.95);
  backdrop-filter: blur(8px);
  padding: 1rem;
}

.lobby-modal {
  width: 100%;
  max-width: 420px;
  padding: 2rem;
  border-radius: 1rem;
  background: linear-gradient(180deg, rgba(31, 41, 55, 0.9) 0%, rgba(15, 23, 42, 0.95) 100%);
  border: 1px solid rgba(148, 163, 184, 0.3);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

.lobby-title {
  margin: 0 0 0.25rem;
  font-size: 2rem;
  font-weight: 700;
  text-align: center;
  color: #f9fafb;
  letter-spacing: 0.08em;
}

.lobby-subtitle {
  margin: 0 0 1.5rem;
  text-align: center;
  color: #94a3b8;
  font-size: 0.95rem;
}

.connection-status {
  margin-bottom: 1rem;
  padding: 0.5rem;
  border-radius: 0.5rem;
  font-size: 0.85rem;
  text-align: center;
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
}

.connection-status--ok {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.connection-status--error {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.lobby-error {
  margin-bottom: 1rem;
  padding: 0.75rem;
  border-radius: 0.5rem;
  background: rgba(239, 68, 68, 0.15);
  color: #fca5a5;
  font-size: 0.9rem;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.lobby-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
}

.tab-btn {
  flex: 1;
  padding: 0.6rem 1rem;
  border: 1px solid rgba(148, 163, 184, 0.4);
  border-radius: 0.5rem;
  background: rgba(15, 23, 42, 0.6);
  color: #94a3b8;
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab-btn:hover {
  background: rgba(30, 41, 59, 0.8);
  color: #e2e8f0;
}

.tab-btn--active {
  background: rgba(34, 197, 94, 0.2);
  border-color: rgba(34, 197, 94, 0.5);
  color: #22c55e;
}

.waiting-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding: 0.5rem 0;
}

.waiting-text {
  margin: 0;
  text-align: center;
  color: #cbd5e1;
  font-size: 0.95rem;
}

.code-display {
  padding: 0.75rem 1.5rem;
  font-family: ui-monospace, monospace;
  font-size: 1.75rem;
  font-weight: 700;
  letter-spacing: 0.25em;
  color: #22c55e;
  background: rgba(34, 197, 94, 0.1);
  border: 2px solid rgba(34, 197, 94, 0.4);
  border-radius: 0.5rem;
  user-select: all;
}

.code-display--copied {
  border-color: rgba(34, 197, 94, 0.8);
  box-shadow: 0 0 12px rgba(34, 197, 94, 0.3);
}

.btn--copy {
  background: linear-gradient(135deg, #0f172a, #1e293b);
  color: #e2e8f0;
  border: 1px solid rgba(148, 163, 184, 0.4);
}

.btn--copy:hover:not(:disabled) {
  background: linear-gradient(135deg, #1e293b, #334155);
}

.waiting-buttons {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  justify-content: center;
}

.join-link-intro {
  margin: 0 0 0.5rem;
  padding: 0.6rem 0.9rem;
  background: rgba(34, 197, 94, 0.1);
  border: 1px solid rgba(34, 197, 94, 0.3);
  border-radius: 0.5rem;
  color: #86efac;
  font-size: 0.9rem;
}

.btn--copied {
  background: rgba(34, 197, 94, 0.25);
  border-color: rgba(34, 197, 94, 0.5);
  color: #22c55e;
}

.waiting-hint {
  margin: 0;
  font-size: 0.85rem;
  color: #64748b;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  50% { opacity: 0.6; }
}

.lobby-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.lobby-label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.9rem;
  color: #cbd5e1;
}

.lobby-input {
  padding: 0.65rem 0.9rem;
  border-radius: 0.5rem;
  border: 1px solid rgba(148, 163, 184, 0.4);
  background: rgba(15, 23, 42, 0.8);
  color: #f9fafb;
  font-size: 1rem;
}

.lobby-input:focus {
  outline: none;
  border-color: rgba(34, 197, 94, 0.6);
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.2);
}

.lobby-input::placeholder {
  color: #64748b;
}

.lobby-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.lobby-input--code {
  font-family: ui-monospace, monospace;
  letter-spacing: 0.15em;
  text-transform: uppercase;
}

.btn {
  padding: 0.75rem 1.5rem;
  border-radius: 999px;
  border: none;
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  transition: transform 0.1s ease, filter 0.15s ease;
}

.btn--primary {
  background: linear-gradient(135deg, #16a34a, #22c55e);
  color: #f9fafb;
  box-shadow: 0 10px 25px rgba(34, 197, 94, 0.35);
  margin-top: 0.25rem;
}

.btn--primary:hover:not(:disabled) {
  filter: brightness(1.1);
}

.btn--primary:active:not(:disabled) {
  transform: translateY(1px);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
