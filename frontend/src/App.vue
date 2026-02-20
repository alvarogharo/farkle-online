<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import Die from '@/components/Die.vue';
import LobbyModal from '@/components/LobbyModal.vue';
import { useWebSocket } from '@/composables/useWebSocket.js';

const ws = useWebSocket();
const inGame = ref(false);
const gameCode = ref('');
const myPlayerIndex = ref(-1);

function applyGameState(data) {
  const ps = data.players || [];
  players.value = ps.map((p) => ({
    name: p.name || 'Jugador',
    total: p.total ?? 0,
    rounds: [], // El backend no envía rounds; los totals ya están actualizados
  }));

  currentPlayerIndex.value = data.currentPlayerIndex ?? 0;
  victoryScore.value = data.victoryScore ?? 2000;
  winnerIndex.value = data.winnerIndex >= 0 ? data.winnerIndex : null;
  finalRoundTriggerIndex.value = data.finalRoundTriggerIndex >= 0 ? data.finalRoundTriggerIndex : null;

  const diceArr = data.dice || [];
  dices.value = diceArr.map((d) => ({ value: d.value ?? 1, held: !!d.held }));

  const selIndices = data.selectedIndices || [];
  selected.value = dices.value.map((_, i) => selIndices.includes(i));

  let rem = data.remainingDiceCount;
  if (rem === undefined || rem === null) {
    rem = dices.value.length === 0 ? 6 : dices.value.filter((d) => !d.held).length;
  } else if (dices.value.length === 0 && rem === 0) {
    rem = 6; // Mano limpia: siguiente tirada con 6 dados
  }
  remainingDiceCount.value = rem;
  turnPoints.value = data.turnPoints ?? 0;

  const moves = data.turnMoves || [];
  turnMoves.value = moves.map((m) => ({
    id: m.id ?? 0,
    values: m.values || [],
    points: m.points ?? 0,
  }));

  isRolling.value = false;
  isTurnEnding.value = false;
  hasRolledThisTurn.value = dices.value.length > 0;
  hasApartadoThisRoll.value =
    dices.value.some((d) => d.held) || (dices.value.length === 0 && turnPoints.value > 0);
  // No borrar statusMessage aquí para preservar farkle, final_round, game_over
}

let unsubscribeGameMessages = () => {};

onMounted(() => {
  ws.connect();
  unsubscribeGameMessages = ws.onMessage((data) => {
    if (!inGame.value) return;
    if (data.type === 'game_state') {
      applyGameState(data);
    } else if (data.type === 'roll_result') {
      if (rollPlaceholderIntervalId) {
        clearInterval(rollPlaceholderIntervalId);
        rollPlaceholderIntervalId = null;
      }
      const diceArr = data.dice || [];
      dices.value = diceArr.map((d) => ({ value: d.value ?? 1, held: !!d.held }));
      selected.value = dices.value.map(() => false);
      hasRolledThisTurn.value = true;
      const elapsed = rollStartTime ? Date.now() - rollStartTime : 0;
      const minRollMs = 1200;
      const settleMs = 500;
      const wait = Math.max(0, minRollMs - elapsed) + settleMs;
      rollAnimationTimeoutId = setTimeout(() => {
        isRolling.value = false;
        rollAnimationTimeoutId = null;
      }, wait);
    } else if (data.type === 'error') {
      if (rollPlaceholderIntervalId) {
        clearInterval(rollPlaceholderIntervalId);
        rollPlaceholderIntervalId = null;
      }
      isRolling.value = false;
      statusKind.value = 'error';
      statusMessage.value = data.message || 'Error';
    } else if (data.type === 'farkle') {
      statusKind.value = 'warn';
      statusMessage.value = data.message || 'Farkle: pierdes los puntos del turno';
      showToast(data.message || 'Farkle: pierdes los puntos del turno', 'warn');
    } else if (data.type === 'hot_dice') {
      showToast(data.message || '¡Mano limpia! Puedes volver a tirar los 6 dados', 'success');
    } else if (data.type === 'turn_changed') {
      showToast(data.message || 'Cambio de turno', 'info');
    } else if (data.type === 'final_round') {
      statusKind.value = 'success';
      statusMessage.value = data.message || 'Ronda final para el otro jugador';
    } else if (data.type === 'game_over') {
      statusKind.value = 'success';
      statusMessage.value = data.message || 'Partida terminada';
    } else if (data.type === 'player_disconnected') {
      statusKind.value = 'success';
      statusMessage.value = data.message || 'El otro jugador se ha desconectado. Ganas la partida.';
    }
  });
});

const onLobbySuccess = ({ gameCode: code, playerIndex }) => {
  gameCode.value = code;
  myPlayerIndex.value = playerIndex;
  inGame.value = true;
};

// Estados de la app: lobby | playing | finished
const appState = computed(() => {
  if (!inGame.value) return 'lobby';
  if (winnerIndex.value !== null) return 'finished';
  return 'playing';
});

function goToLobby() {
  inGame.value = false;
  gameCode.value = '';
  myPlayerIndex.value = -1;
  resetGame();
}

const players = ref([
  { name: 'Jugador 1', total: 0, rounds: [] },
  { name: 'Jugador 2', total: 0, rounds: [] },
]);
const currentPlayerIndex = ref(0);
const victoryScore = ref(2000);
const winnerIndex = ref(null);
const finalRoundTriggerIndex = ref(null);

// Dados visibles en mesa durante la tirada actual:
// - held=true: ya apartado (no se vuelve a tirar)
// - held=false: dado activo (se vuelve a tirar al pulsar "Tirar")
const dices = ref([]);
const remainingDiceCount = ref(6);
const isRolling = ref(false);
const selected = ref([]);
const hasApartadoThisRoll = ref(false);
const hasRolledThisTurn = ref(false);
const isTurnEnding = ref(false);

const turnPoints = ref(0);
const turnMoves = ref([]);
const statusMessage = ref('');
const statusKind = ref('info'); // info | error | success | warn

const toast = ref({ show: false, message: '', kind: 'info' });
let toastTimeoutId = null;
function showToast(message, kind = 'info') {
  if (toastTimeoutId) clearTimeout(toastTimeoutId);
  toast.value = { show: true, message, kind };
  toastTimeoutId = setTimeout(() => {
    toast.value.show = false;
    toastTimeoutId = null;
  }, 2000);
}

let rollAnimationTimeoutId = null;
let rollPlaceholderIntervalId = null;
let rollStartTime = null;
const clearRollingTimers = () => {
  if (rollAnimationTimeoutId) {
    clearTimeout(rollAnimationTimeoutId);
    rollAnimationTimeoutId = null;
  }
  if (rollPlaceholderIntervalId) {
    clearInterval(rollPlaceholderIntervalId);
    rollPlaceholderIntervalId = null;
  }
};

const isMyTurn = computed(() =>
  inGame.value
  && myPlayerIndex.value >= 0
  && currentPlayerIndex.value === myPlayerIndex.value,
);

const rollDices = () => {
  if (!isMyTurn.value) return;
  if (isRolling.value) return;
  if (winnerIndex.value !== null) return;
  if (dices.value.length > 0 && !hasApartadoThisRoll.value) return;

  isRolling.value = true;
  rollStartTime = Date.now();
  statusMessage.value = '';
  const count = dices.value.length || remainingDiceCount.value;
  if (dices.value.length === 0) {
    // Primera tirada: mostrar dados con valores que cambian (animación de lanzamiento)
    dices.value = Array.from({ length: count }, () => ({
      value: Math.floor(Math.random() * 6) + 1,
      held: false,
    }));
    selected.value = dices.value.map(() => false);
    const rollPlaceholder = () => {
      dices.value = dices.value.map((d) => ({
        ...d,
        value: Math.floor(Math.random() * 6) + 1,
      }));
    };
    rollPlaceholderIntervalId = setInterval(rollPlaceholder, 80);
  }
  // Si ya hay dados (re-tirada), solo se sacuden, no cambian de valor hasta roll_result
  ws.send({ type: 'roll' });
};

const toggleSelect = (index) => {
  if (!isMyTurn.value) return;
  if (isTurnEnding.value || isRolling.value) return;
  if (winnerIndex.value !== null) return;
  if (!hasRolledThisTurn.value || !dices.value.length) return;
  if (dices.value[index]?.held) return;
  ws.send({ type: 'toggle_select', index });
};

const hasSelection = computed(() => selected.value.some((v) => v));

const apartarSeleccionados = () => {
  if (!hasSelection.value || !isMyTurn.value) return;
  if (isRolling.value || winnerIndex.value !== null) return;
  if (!hasRolledThisTurn.value || !dices.value.length) return;
  ws.send({ type: 'apartar' });
};

// Regla: no se puede "plantarse" con una tirada sin haber apartado al menos una vez
const canBank = computed(() =>
  turnPoints.value > 0
  && !isRolling.value
  && winnerIndex.value === null
  && (dices.value.length === 0 || hasApartadoThisRoll.value),
);

const bankTurn = () => {
  if (!canBank.value || !isMyTurn.value) return;
  ws.send({ type: 'bank' });
};

const resetGame = () => {
  clearRollingTimers();
  isRolling.value = false;
  players.value = [
    { name: 'Jugador 1', total: 0, rounds: [] },
    { name: 'Jugador 2', total: 0, rounds: [] },
  ];
  currentPlayerIndex.value = 0;
  winnerIndex.value = null;
  turnPoints.value = 0;
  turnMoves.value = [];
  remainingDiceCount.value = 6;
  dices.value = [];
  selected.value = [];
  hasApartadoThisRoll.value = false;
  hasRolledThisTurn.value = false;
  finalRoundTriggerIndex.value = null;
  isTurnEnding.value = false;
  statusMessage.value = '';
  statusKind.value = 'info';
};

onBeforeUnmount(() => {
  clearRollingTimers();
  if (toastTimeoutId) clearTimeout(toastTimeoutId);
  unsubscribeGameMessages();
});
</script>

<template>
  <LobbyModal
    v-if="!inGame"
    :send="ws.send"
    :on-message="ws.onMessage"
    :connected="ws.connected.value"
    :last-error="ws.lastError?.value ?? null"
    @success="onLobbySuccess"
  />
  <main v-show="inGame" class="page">
    <h1>Farkle</h1>
    <button
      v-if="appState === 'finished'"
      type="button"
      class="btn btn--secondary"
      @click="goToLobby"
    >
      Volver al lobby
    </button>

    <!-- Overlay de partida terminada -->
    <div v-if="appState === 'finished'" class="game-over-overlay">
      <div class="game-over-card">
        <h2 class="game-over-title">Partida terminada</h2>
        <p class="game-over-winner">
          {{ players[winnerIndex].name }} gana con {{ players[winnerIndex].total }} puntos
        </p>
        <button
          type="button"
          class="btn btn--primary"
          @click="goToLobby"
        >
          Volver al lobby
        </button>
      </div>
    </div>
    
    <section class="scoreboard">
      <div
        v-for="(p, idx) in players"
        :key="p.name"
        class="player-card"
        :class="{
          'player-card--active': idx === currentPlayerIndex && winnerIndex === null,
          'player-card--winner': idx === winnerIndex,
        }"
      >
        <div class="player-name">
          {{ p.name }}
        </div>
        <div class="player-total">
          Total: {{ p.total }}
        </div>
        <div class="player-rounds">
          Rondas:
          <span v-if="!p.rounds.length" class="muted">—</span>
          <span
            v-for="r in p.rounds"
            :key="r.id"
            class="round-chip"
          >
            +{{ r.points }}
          </span>
        </div>
      </div>
    </section>

    <section class="help-section">
      <h2>Combinaciones y puntos</h2>
      <div class="help-table">
        <div class="help-col">
          <div class="help-row">5 = 50 pts</div>
          <div class="help-row">1 = 100 pts</div>
          <div class="help-row">Tres 2 = 200 pts</div>
          <div class="help-row">Tres 3 = 300 pts</div>
          <div class="help-row">Tres 4 = 400 pts</div>
          <div class="help-row">Tres 5 = 500 pts</div>
          <div class="help-row">Tres 6 = 600 pts</div>
          <div class="help-row">Tres 1 = 1000 pts</div>
        </div>
        <div class="help-col">
          <div class="help-row">Cuatro iguales de cualquier número = 1000 pts</div>
          <div class="help-row">Escalera 1–6 = 1500 pts</div>
          <div class="help-row">Tres parejas = 1500 pts</div>
          <div class="help-row">Cuatro iguales y una pareja = 1500 pts</div>
          <div class="help-row">Cinco iguales de cualquier número = 2000 pts</div>
          <div class="help-row">Dos tríos = 2500 pts</div>
          <div class="help-row">Seis iguales de cualquier número = 3000 pts</div>
        </div>
      </div>
    </section>

    <section class="controls">
      <h2>
        Turno: {{ players[currentPlayerIndex].name }} · Puntos del turno: {{ turnPoints }}
      </h2>

      <div
        v-if="statusMessage"
        class="status"
        :class="`status--${statusKind}`"
      >
        {{ statusMessage }}
      </div>
    </section>

    <section class="dice-section">
      <div class="dice-row">
        <template v-if="dices.length">
          <Die
            v-for="(die, index) in dices"
            :key="index"
            :value="die.value"
            :is-rolling="isRolling"
            :selected="selected[index] && !die.held"
            :disabled="!isMyTurn || winnerIndex !== null || die.held || !hasRolledThisTurn"
            :held="die.held"
            @toggle="toggleSelect(index)"
          />
        </template>
        <div
          v-else
          class="dice-empty"
        >
          Tienes {{ remainingDiceCount }} dados listos. Pulsa “Tirar dados”.
        </div>
      </div>
    </section>

    <section class="action-bar">
      <button
        type="button"
        class="btn"
        :disabled="!isMyTurn || isRolling || winnerIndex !== null || (dices.length > 0 && !hasApartadoThisRoll)"
        @click="rollDices"
      >
        {{ isRolling ? 'Tirando...' : 'Tirar dados' }}
      </button>

      <button
        type="button"
        class="btn btn--secondary"
        :disabled="!isMyTurn || isRolling || !hasSelection || winnerIndex !== null || !hasRolledThisTurn"
        @click="apartarSeleccionados"
      >
        Apartar
      </button>

      <button
        type="button"
        class="btn btn--bank"
        :disabled="!canBank || !isMyTurn"
        @click="bankTurn"
      >
        Plantarse
      </button>
    </section>

    <section
      v-if="turnMoves.length"
      class="saved-section"
    >
      <h3>Apartados del turno</h3>
      <div
        v-for="move in turnMoves"
        :key="move.id"
        class="saved-group"
      >
        <span class="saved-label">Lote {{ move.id }}:</span>
        <div class="saved-dice-list">
          <Die
            v-for="(valor, idx) in move.values"
            :key="idx"
            :value="valor"
            :is-rolling="false"
            :selected="false"
            :compact="true"
            :disabled="true"
            class="saved-die"
          />
        </div>
        <div class="saved-points">
          +{{ move.points }}
        </div>
      </div>
    </section>

    <!-- Toast para Farkle, Hot dice, cambio de turno -->
    <Transition name="toast">
      <div
        v-if="toast.show"
        class="toast"
        :class="`toast--${toast.kind}`"
      >
        {{ toast.message }}
      </div>
    </Transition>
  </main>
</template>

<style scoped>
.page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  gap: 2rem;
  padding: 2rem 1rem;
  background: radial-gradient(circle at top, #1f2933 0, #050816 50%, #02040b 100%);
  color: #f9fafb;
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

h1 {
  font-size: 2.5rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  text-shadow: 0 0 15px rgba(0, 0, 0, 0.6);
}

.game-over-overlay {
  position: fixed;
  inset: 0;
  z-index: 500;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(2, 4, 11, 0.9);
  backdrop-filter: blur(6px);
}

.game-over-card {
  padding: 2rem 2.5rem;
  border-radius: 1rem;
  background: linear-gradient(180deg, rgba(31, 41, 55, 0.95) 0%, rgba(15, 23, 42, 0.98) 100%);
  border: 1px solid rgba(148, 163, 184, 0.3);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
  text-align: center;
}

.game-over-title {
  margin: 0 0 0.75rem;
  font-size: 1.5rem;
  color: #f9fafb;
}

.game-over-winner {
  margin: 0 0 1.5rem;
  font-size: 1.25rem;
  color: #22c55e;
  font-weight: 600;
}

.toast {
  position: fixed;
  bottom: 2rem;
  left: 50%;
  transform: translateX(-50%);
  z-index: 400;
  padding: 1rem 2rem;
  border-radius: 0.75rem;
  font-size: 1.1rem;
  font-weight: 600;
  text-align: center;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
  pointer-events: none;
}

.toast--warn {
  background: rgba(245, 158, 11, 0.95);
  color: #1f2937;
  border: 2px solid rgba(251, 191, 36, 0.8);
}

.toast--success {
  background: rgba(34, 197, 94, 0.95);
  color: #f9fafb;
  border: 2px solid rgba(74, 222, 128, 0.8);
}

.toast--info {
  background: rgba(59, 130, 246, 0.95);
  color: #f9fafb;
  border: 2px solid rgba(96, 165, 250, 0.8);
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(1rem);
}

.controls {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.scoreboard {
  width: 100%;
  max-width: 820px;
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
}

.player-card {
  padding: 0.75rem;
  border-radius: 0.75rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.55);
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.player-card--active {
  border-color: rgba(34, 197, 94, 0.65);
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.25);
}

.player-card--winner {
  border-color: rgba(59, 130, 246, 0.7);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}

.player-name {
  font-weight: 700;
}

.player-total {
  color: #e5e7eb;
}

.player-rounds {
  font-size: 0.9rem;
  color: #cbd5e1;
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  align-items: center;
}

.round-chip {
  font-size: 0.8rem;
  padding: 0.12rem 0.4rem;
  border-radius: 999px;
  background: rgba(31, 41, 55, 0.85);
  border: 1px solid rgba(75, 85, 99, 0.7);
}

.muted {
  color: #9ca3af;
}

.help-section {
  width: 100%;
  max-width: 900px;
  background: rgba(15, 23, 42, 0.6);
  border-radius: 0.75rem;
  padding: 0.75rem 1rem;
  border: 1px solid rgba(148, 163, 184, 0.4);
}

.help-section h2 {
  font-size: 0.95rem;
  margin-bottom: 0.4rem;
}

.help-table {
  display: flex;
  gap: 1rem;
  font-size: 0.8rem;
}

.help-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.help-row {
  white-space: nowrap;
}


.btn {
  padding: 0.75rem 1.5rem;
  border-radius: 999px;
  border: none;
  background: linear-gradient(135deg, #16a34a, #22c55e);
  color: #f9fafb;
  font-weight: 600;
  font-size: 1rem;
  cursor: pointer;
  box-shadow: 0 10px 25px rgba(34, 197, 94, 0.35);
  transition: transform 0.1s ease, box-shadow 0.1s ease, filter 0.1s ease;
}

.btn--secondary {
  background: linear-gradient(135deg, #0f172a, #1f2937);
  box-shadow: 0 10px 25px rgba(15, 23, 42, 0.65);
}

.btn--bank {
  background: linear-gradient(135deg, #2563eb, #3b82f6);
  box-shadow: 0 10px 25px rgba(59, 130, 246, 0.35);
}

.btn:hover {
  filter: brightness(1.1);
  box-shadow: 0 14px 30px rgba(34, 197, 94, 0.45);
}

.btn:active {
  transform: translateY(1px) scale(0.99);
  box-shadow: 0 8px 18px rgba(34, 197, 94, 0.35);
}

.btn:disabled {
  opacity: 0.6;
  cursor: default;
  box-shadow: 0 6px 14px rgba(15, 23, 42, 0.6);
}

.dice-section {
  display: flex;
  justify-content: center;
  width: 100%;
}

.dice-row {
  display: flex;
  flex-wrap: wrap;
  gap: 1.5rem;
  justify-content: center;
}

.action-bar {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 0.75rem;
  width: 100%;
}

.dice-empty {
  color: #cbd5e1;
  padding: 0.75rem 1rem;
  border-radius: 0.75rem;
  border: 1px dashed rgba(148, 163, 184, 0.45);
  background: rgba(15, 23, 42, 0.35);
}

.saved-section {
  margin-top: 1.5rem;
  width: 100%;
  max-width: 640px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.saved-section h3 {
  font-size: 1.1rem;
}

.saved-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border-radius: 0.5rem;
  background: rgba(15, 23, 42, 0.7);
  border: 1px solid rgba(148, 163, 184, 0.4);
}

.saved-label {
  font-size: 0.85rem;
  color: #9ca3af;
}

.saved-dice-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.saved-die {
  cursor: default;
}

.saved-points {
  margin-left: auto;
  font-weight: 700;
  color: #e5e7eb;
}

.status {
  margin-top: 0.25rem;
  padding: 0.5rem 0.75rem;
  border-radius: 0.5rem;
  border: 1px solid rgba(148, 163, 184, 0.35);
  background: rgba(15, 23, 42, 0.55);
  max-width: 720px;
  text-align: center;
}

.status--error {
  border-color: rgba(239, 68, 68, 0.55);
}

.status--warn {
  border-color: rgba(245, 158, 11, 0.55);
}

.status--success {
  border-color: rgba(34, 197, 94, 0.55);
}
</style>
