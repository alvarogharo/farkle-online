<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue';
import {
  DEFAULT_VICTORY_SCORE,
  MIN_VICTORY_SCORE,
  MAX_VICTORY_SCORE,
  MSG,
  MSG_LOBBY,
  MSG_SEND,
} from '@/config.js';
import { lobby as lobbyMsg } from '@/messages.js';

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
const joinedPlayers = ref([]);
const pendingGameCode = ref('');
const joinViaLink = ref(false); // true cuando llegas con ?code=XXX
const localPlayerIndex = ref(0);
const localGameCode = ref('');

// Crear partida
const createName = ref('');
const createVictoryScore = ref(DEFAULT_VICTORY_SCORE);
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
  if (data.type === MSG_LOBBY.ERROR) {
    serverError.value = data.message || lobbyMsg.serverError;
    createLoading.value = false;
    joinLoading.value = false;
    return;
  }
  if (data.type === MSG_LOBBY.GAME_CREATED) {
    createLoading.value = false;
    serverError.value = '';
    pendingGameCode.value = data.gameCode || '';
    localGameCode.value = pendingGameCode.value;
    localPlayerIndex.value = 0;
    waitingForPlayer.value = true;
    return;
  }
  if (data.type === MSG_LOBBY.GAME_JOINED) {
    joinLoading.value = false;
    waitingForPlayer.value = true;
    pendingGameCode.value = data.gameCode;
    localGameCode.value = data.gameCode;
    localPlayerIndex.value = data.playerIndex ?? 1;
    return;
  }
  if (data.type === MSG.GAME_STATE && waitingForPlayer.value) {
    const ps = data.players || [];
    joinedPlayers.value = ps
      .map((p, idx) => ({
        index: idx,
        name: p.name || '',
        active: p.active,
      }))
      .filter((p) => p.active && p.name);
    return;
  }
  if (data.type === MSG_LOBBY.GAME_STARTED) {
    const code = localGameCode.value || pendingGameCode.value || data.gameCode || '';
    emit('success', { gameCode: code, playerIndex: localPlayerIndex.value });
    return;
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
  const raw = createName.value.trim();
  const len = raw.length;
  if (len < 1 || len > 16) {
    serverError.value = len < 1
      ? lobbyMsg.nameRequired
      : lobbyMsg.nameTooLong;
    return;
  }

  const name = capitalizeName(raw) || lobbyMsg.defaultPlayer1;
  const victoryScore = Math.max(
    MIN_VICTORY_SCORE,
    Math.min(MAX_VICTORY_SCORE, Number(createVictoryScore.value) || DEFAULT_VICTORY_SCORE),
  );
  createLoading.value = true;
  props.send({ type: MSG_SEND.CREATE, playerName: name, victoryScore });
}

function doJoin() {
  clearErrors();
  const raw = joinName.value.trim();
  const len = raw.length;
  if (len < 1 || len > 16) {
    serverError.value = len < 1
      ? lobbyMsg.nameRequired
      : lobbyMsg.nameTooLong;
    return;
  }

  const name = capitalizeName(raw) || lobbyMsg.defaultPlayer2;
  const code = joinCode.value.trim().toUpperCase();
  if (!code) {
    serverError.value = lobbyMsg.codeRequired;
    return;
  }
  joinLoading.value = true;
  props.send({ type: MSG_SEND.JOIN, gameCode: code, playerName: name });
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
      <p class="lobby-subtitle">{{ lobbyMsg.subtitle }}</p>

      <div class="connection-status" :class="{ 'connection-status--ok': connected, 'connection-status--error': lastError }">
        <span v-if="connected">{{ lobbyMsg.connected }}</span>
        <span v-else-if="lastError">{{ lobbyMsg.connectionError }}</span>
        <span v-else>{{ lobbyMsg.connecting }}</span>
      </div>

      <div v-if="serverError" class="lobby-error">
        {{ serverError }}
      </div>

      <!-- Pantalla de espera tras crear partida -->
      <div v-if="waitingForPlayer" class="waiting-section">
        <p class="waiting-text">{{ lobbyMsg.shareHint }}</p>
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
            {{ copyFeedback ? lobbyMsg.copyCodeDone : lobbyMsg.copyCode }}
          </button>
          <button
            type="button"
            class="btn btn--copy"
            :class="{ 'btn--copied': linkCopyFeedback }"
            @click="copyLink"
          >
            {{ linkCopyFeedback ? lobbyMsg.copyLinkDone : lobbyMsg.copyLink }}
          </button>
        </div>

        <div v-if="joinedPlayers.length" class="waiting-players">
          <h2 class="waiting-players__title">Players in room</h2>
          <ul class="waiting-players__list">
            <li
              v-for="p in joinedPlayers"
              :key="p.index"
              class="waiting-players__item"
            >
              {{ p.name }}
            </li>
          </ul>
          <button
            v-if="localPlayerIndex === 0"
            type="button"
            class="btn btn--primary"
            :disabled="!connected"
            @click="props.send({ type: MSG_SEND.START, gameCode: pendingGameCode })"
          >
            Start game
          </button>
        </div>

        <p v-else class="waiting-hint">{{ lobbyMsg.waitingHint }}</p>
      </div>

      <template v-else>
      <div class="lobby-tabs">
        <button
          type="button"
          class="tab-btn"
          :class="{ 'tab-btn--active': tab === 'create' }"
          @click="tab = 'create'; clearErrors()"
        >
          {{ lobbyMsg.createGame }}
        </button>
        <button
          type="button"
          class="tab-btn"
          :class="{ 'tab-btn--active': tab === 'join' }"
          @click="tab = 'join'; clearErrors()"
        >
          {{ lobbyMsg.joinGame }}
        </button>
      </div>

      <form v-if="tab === 'create'" class="lobby-form" @submit.prevent="doCreate">
        <label class="lobby-label">
          {{ lobbyMsg.name }}
          <input
            v-model="createName"
            type="text"
            class="lobby-input"
            :placeholder="lobbyMsg.defaultPlayer1"
            :disabled="!connected"
          >
        </label>
        <label class="lobby-label">
          {{ lobbyMsg.scoreToWin }}
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
          {{ createLoading ? lobbyMsg.creating : lobbyMsg.createGame }}
        </button>
      </form>

      <form v-else class="lobby-form" @submit.prevent="doJoin">
        <p v-if="joinViaLink" class="join-link-intro">
          {{ lobbyMsg.joinLinkIntro }}
        </p>
        <label class="lobby-label">
          {{ lobbyMsg.name }}
          <input
            v-model="joinName"
            type="text"
            class="lobby-input"
            :placeholder="joinViaLink ? lobbyMsg.yourName : lobbyMsg.defaultPlayer2"
            :disabled="!connected"
          >
        </label>
        <label v-if="!joinViaLink" class="lobby-label">
          {{ lobbyMsg.gameCode }}
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
          {{ joinLoading ? lobbyMsg.joining : lobbyMsg.join }}
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
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
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
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
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

.waiting-players {
  margin-top: 1rem;
  width: 100%;
  max-width: 260px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.waiting-players__title {
  margin: 0 0 0.4rem;
  font-size: 0.95rem;
  color: #f9fafb;
  font-weight: 600;
}

.waiting-players__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-bottom: 0.75rem;
}

.waiting-players__item {
  font-size: 0.9rem;
  color: #f9fafb;
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.waiting-players__item::before {
  content: counter(player-item) '.';
}

.waiting-players__list {
  counter-reset: player-item;
}

.waiting-players__item {
  counter-increment: player-item;
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
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
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
