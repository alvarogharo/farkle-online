<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import Die from '@/components/Die.vue';
import LobbyModal from '@/components/LobbyModal.vue';
import { useGameState } from '@/composables/useGameState.js';
import { useWebSocket } from '@/composables/useWebSocket.js';
import {
  DICE_SIDES,
  MSG,
  MSG_SEND,
} from '@/config.js';
import { game as msg } from '@/messages.js';

const ws = useWebSocket();
const inGame = ref(false);
const gameCode = ref('');
const myPlayerIndex = ref(-1);

const game = useGameState(myPlayerIndex);

const {
  players,
  currentPlayerIndex,
  victoryScore,
  winnerIndex,
  finalRoundTriggerIndex,
  finishedByDisconnect,
  dices,
  remainingDiceCount,
  isRolling,
  selected,
  hasApartadoThisRoll,
  hasRolledThisTurn,
  isTurnEnding,
  turnPoints,
  turnMoves,
  statusMessage,
  statusKind,
  farklePendingTransition,
  pendingGameState,
  rollResultPending,
  rollResultTime,
  timerIds,
  applyGameState,
  clearRollingTimers,
  resetGame,
  ROLL_DISPLAY_MS,
  ROLL_ANIMATION_INTERVAL_MS,
  TOAST_DELAY_MS,
  TOAST_DURATION_MS,
} = game;

let unsubscribeGameMessages = () => {};

onMounted(() => {
  ws.connect();
  unsubscribeGameMessages = ws.onMessage((data) => {
    if (!inGame.value) return;
    if (data.type === MSG.GAME_STATE) {
      if (farklePendingTransition.value) {
        pendingGameState.value = data;
        const baseTime = rollResultTime.value ?? 0;
        const toastDisappearAt = baseTime
          ? baseTime + ROLL_DISPLAY_MS + TOAST_DELAY_MS + TOAST_DURATION_MS
          : Date.now() + TOAST_DURATION_MS;
        const delayMs = Math.max(0, toastDisappearAt - Date.now());
        timerIds.farkleDelayTimeoutId = setTimeout(() => {
          applyGameState(pendingGameState.value, { preserveDice: true });
          pendingGameState.value = null;
          farklePendingTransition.value = false;
          timerIds.farkleDelayTimeoutId = null;
        }, delayMs);
      } else {
        const newPlayerIdx = data.currentPlayerIndex ?? 0;
        const isTurnChange = newPlayerIdx !== currentPlayerIndex.value;
        const isTurnOrFarkleTransition =
          (!data.dice || data.dice.length === 0) && dices.value.length > 0 && isTurnChange;
        applyGameState(data, { preserveDice: isTurnOrFarkleTransition });
      }
    } else if (data.type === MSG.ROLL_RESULT) {
      if (timerIds.rollPlaceholderIntervalId) {
        clearInterval(timerIds.rollPlaceholderIntervalId);
        timerIds.rollPlaceholderIntervalId = null;
      }
      const diceArr = data.dice || [];
      const realDice = diceArr.map((d) => ({ value: d.value ?? 1, held: !!d.held }));
      rollResultTime.value = Date.now();
      selected.value = realDice.map(() => false);
      hasRolledThisTurn.value = true;
      rollResultPending.value = true;
      isRolling.value = true;
      // Mostrar dados con valores aleatorios que cambian durante la animación (solo los no held)
      dices.value = realDice.map((d) =>
        d.held ? d : { ...d, value: Math.floor(Math.random() * DICE_SIDES) + 1 },
      );
      const rollDisplayInterval = () => {
        dices.value = dices.value.map((d) =>
          d.held ? d : { ...d, value: Math.floor(Math.random() * DICE_SIDES) + 1 },
        );
      };
      timerIds.rollPlaceholderIntervalId = setInterval(rollDisplayInterval, ROLL_ANIMATION_INTERVAL_MS);
      timerIds.rollAnimationTimeoutId = setTimeout(() => {
        clearInterval(timerIds.rollPlaceholderIntervalId);
        timerIds.rollPlaceholderIntervalId = null;
        dices.value = realDice;
        isRolling.value = false;
        rollResultPending.value = false;
        timerIds.rollAnimationTimeoutId = null;
      }, ROLL_DISPLAY_MS);
    } else if (data.type === MSG.ERROR) {
      if (timerIds.rollPlaceholderIntervalId) {
        clearInterval(timerIds.rollPlaceholderIntervalId);
        timerIds.rollPlaceholderIntervalId = null;
      }
      if (timerIds.rollAnimationTimeoutId) {
        clearTimeout(timerIds.rollAnimationTimeoutId);
        timerIds.rollAnimationTimeoutId = null;
      }
      if (timerIds.farkleToastTimeoutId) {
        clearTimeout(timerIds.farkleToastTimeoutId);
        timerIds.farkleToastTimeoutId = null;
      }
      rollResultPending.value = false;
      isRolling.value = false;
      statusKind.value = 'error';
      statusMessage.value = data.message || msg.errorDefault;
    } else if (data.type === MSG.FARKLE) {
      const farklePlayerName = players.value[currentPlayerIndex.value]?.name || msg.playerDefault;
      const farkleMsg = myPlayerIndex.value === currentPlayerIndex.value
        ? msg.farkleSelf
        : msg.farkleOther(farklePlayerName);
      farklePendingTransition.value = true;
      statusMessage.value = '';
      const elapsed = rollResultTime.value ? Date.now() - rollResultTime.value : ROLL_DISPLAY_MS;
      const waitForAnimation = Math.max(0, ROLL_DISPLAY_MS - elapsed) + TOAST_DELAY_MS;
      timerIds.farkleToastTimeoutId = setTimeout(() => {
        showToast(farkleMsg, 'warn');
        timerIds.farkleToastTimeoutId = null;
      }, waitForAnimation);
    } else if (data.type === MSG.HOT_DICE) {
      const hotDicePlayerName = players.value[currentPlayerIndex.value]?.name || msg.playerDefault;
      const hotDiceMsg = myPlayerIndex.value === currentPlayerIndex.value
        ? msg.hotDiceSelf
        : msg.hotDiceOther(hotDicePlayerName);
      showToast(hotDiceMsg, 'success');
    } else if (data.type === MSG.TURN_CHANGED) {
      const nextIdx = 1 - currentPlayerIndex.value;
      const turnMsg = myPlayerIndex.value === nextIdx
        ? msg.turnYourTurn
        : (data.message || msg.turnChanged);
      showToast(turnMsg, 'info');
    } else if (data.type === MSG.FINAL_ROUND) {
      statusKind.value = 'success';
      const iTriggered = myPlayerIndex.value === currentPlayerIndex.value;
      const finalMsg = iTriggered
        ? msg.finalRoundTriggered
        : msg.finalRoundLastTurn;
      statusMessage.value = finalMsg;
    } else if (data.type === MSG.GAME_OVER) {
      statusKind.value = 'success';
      const winnerIdx = data.winner;
      const winnerName = winnerIdx >= 0 && players.value[winnerIdx] ? players.value[winnerIdx].name : null;
      const gameOverMsg = winnerName && myPlayerIndex.value === winnerIdx
        ? msg.gameOverYouWin
        : (winnerName ? msg.gameOverOtherWins(winnerName) : msg.gameOverFinished);
      statusMessage.value = gameOverMsg;
    } else if (data.type === MSG.PLAYER_DISCONNECTED) {
      finishedByDisconnect.value = true;
      statusKind.value = 'success';
      statusMessage.value = data.message || msg.disconnectYouWin;
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
  finishedByDisconnect.value = false;
  resetGame();
  if (window.location.search) {
    const url = new URL(window.location.href);
    url.search = '';
    window.history.replaceState({}, '', url.pathname + url.hash);
  }
}

const toast = ref({ show: false, message: '', kind: 'info' });
let toastTimeoutId = null;
function showToast(message, kind = 'info') {
  if (toastTimeoutId) clearTimeout(toastTimeoutId);
  toast.value = { show: true, message, kind };
  toastTimeoutId = setTimeout(() => {
    toast.value.show = false;
    toastTimeoutId = null;
  }, TOAST_DURATION_MS);
}

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
  statusMessage.value = '';
  const count = dices.value.length || remainingDiceCount.value;
  if (dices.value.length === 0) {
    // Primera tirada: mostrar dados con valores que cambian (animación de lanzamiento)
    dices.value = Array.from({ length: count }, () => ({
      value: Math.floor(Math.random() * DICE_SIDES) + 1,
      held: false,
    }));
    selected.value = dices.value.map(() => false);
    const rollPlaceholder = () => {
      dices.value = dices.value.map((d) => ({
        ...d,
        value: Math.floor(Math.random() * DICE_SIDES) + 1,
      }));
    };
    timerIds.rollPlaceholderIntervalId = setInterval(rollPlaceholder, ROLL_ANIMATION_INTERVAL_MS);
  }
  // Si ya hay dados (re-tirada), solo se sacuden, no cambian de valor hasta roll_result
  ws.send({ type: MSG_SEND.ROLL });
};

const toggleSelect = (index) => {
  if (!isMyTurn.value) return;
  if (isTurnEnding.value || isRolling.value) return;
  if (winnerIndex.value !== null) return;
  if (!hasRolledThisTurn.value || !dices.value.length) return;
  if (dices.value[index]?.held) return;
  if (statusKind.value === 'error') {
    statusMessage.value = '';
    statusKind.value = 'info';
  }
  ws.send({ type: MSG_SEND.TOGGLE_SELECT, index });
};

const hasSelection = computed(() => selected.value.some((v) => v));

const apartarSeleccionados = () => {
  if (!hasSelection.value || !isMyTurn.value) return;
  if (isRolling.value || winnerIndex.value !== null) return;
  if (!hasRolledThisTurn.value || !dices.value.length) return;
  ws.send({ type: MSG_SEND.APARTAR });
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
  ws.send({ type: MSG_SEND.BANK });
};

onBeforeUnmount(() => {
  clearRollingTimers();
  if (toastTimeoutId) clearTimeout(toastTimeoutId);
  if (timerIds.farkleDelayTimeoutId) clearTimeout(timerIds.farkleDelayTimeoutId);
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
      {{ msg.backToLobby }}
    </button>

    <!-- Overlay de partida terminada -->
    <div v-if="appState === 'finished'" class="game-over-overlay">
      <div class="game-over-card">
        <h2 class="game-over-title">{{ msg.gameOverTitle }}</h2>
        <template v-if="finishedByDisconnect">
          <p class="game-over-result game-over-result--disconnect">
            {{ msg.disconnectOverlay }}
          </p>
        </template>
        <template v-else>
          <p
            class="game-over-result"
            :class="{ 'game-over-result--win': myPlayerIndex === winnerIndex, 'game-over-result--lose': myPlayerIndex !== winnerIndex }"
          >
            {{ myPlayerIndex === winnerIndex ? msg.youWin : msg.youLose }}
          </p>
        </template>
        <div class="game-over-scores">
          <p
            v-for="(p, idx) in players"
            :key="p.name"
            class="game-over-score"
            :class="{ 'game-over-score--winner': idx === winnerIndex }"
          >
            {{ p.name }}: {{ p.total }} puntos
          </p>
        </div>
        <button
          type="button"
          class="btn btn--primary"
          @click="goToLobby"
        >
          {{ msg.backToLobby }}
        </button>
      </div>
    </div>
    
    <section class="scoreboard">
      <div
        v-if="finalRoundTriggerIndex !== null && winnerIndex === null"
        class="final-round-banner"
      >
        <span class="final-round-banner__icon">🏁</span>
        ¡Ronda final! {{ finalRoundTriggerIndex === myPlayerIndex ? msg.finalRoundBannerTriggered : msg.finalRoundBannerOther(players[1 - finalRoundTriggerIndex]?.name || msg.playerDefault) }}
      </div>
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
        {{ isMyTurn ? msg.yourTurn : msg.turnOf(players[currentPlayerIndex]?.name || msg.playerDefault) }} · Puntos del turno: {{ turnPoints }}
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
            :is-rolling="isRolling && !die.held"
            :selected="selected[index] && !die.held"
            :disabled="farklePendingTransition || !isMyTurn || winnerIndex !== null || die.held || !hasRolledThisTurn"
            :held="die.held"
            @toggle="toggleSelect(index)"
          />
        </template>
        <div
          v-else
          class="dice-empty"
        >
          <template v-if="isMyTurn">
            {{ msg.diceReady(remainingDiceCount) }}
          </template>
          <template v-else>
            {{ msg.waitingForTurn(players[currentPlayerIndex]?.name || msg.playerDefault) }}
          </template>
        </div>
      </div>
    </section>

    <section class="action-bar">
      <button
        type="button"
        class="btn"
        :disabled="farklePendingTransition || !isMyTurn || isRolling || winnerIndex !== null || (dices.length > 0 && !hasApartadoThisRoll)"
        @click="rollDices"
      >
        {{ isRolling ? msg.rolling : msg.rollDice }}
      </button>

      <button
        type="button"
        class="btn btn--secondary"
        :disabled="farklePendingTransition || !isMyTurn || isRolling || !hasSelection || winnerIndex !== null || !hasRolledThisTurn"
        @click="apartarSeleccionados"
      >
        {{ msg.apartar }}
      </button>

      <button
        type="button"
        class="btn btn--bank"
        :disabled="farklePendingTransition || !canBank || !isMyTurn"
        @click="bankTurn"
      >
        {{ msg.bank }}
      </button>
    </section>

    <section
      v-if="turnMoves.length"
      class="saved-section"
    >
      <h3>{{ msg.savedSection }}</h3>
      <div
        v-for="move in turnMoves"
        :key="move.id"
        class="saved-group"
      >
        <span class="saved-label">{{ msg.lotLabel(move.id) }}:</span>
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

.game-over-result {
  margin: 0 0 1rem;
  font-size: 1.4rem;
  font-weight: 700;
}

.game-over-result--win {
  color: #22c55e;
}

.game-over-result--lose {
  color: #f87171;
}

.game-over-result--disconnect {
  color: #94a3b8;
}

.game-over-scores {
  margin: 0 0 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 1rem;
  color: #cbd5e1;
}

.game-over-score {
  margin: 0;
}

.game-over-score--winner {
  color: #22c55e;
  font-weight: 600;
}

.toast {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
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
  transform: translate(-50%, -50%) scale(0.9);
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

.final-round-banner {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.6rem 1rem;
  background: linear-gradient(135deg, rgba(234, 179, 8, 0.2), rgba(202, 138, 4, 0.25));
  border: 1px solid rgba(234, 179, 8, 0.5);
  border-radius: 0.5rem;
  font-weight: 600;
  color: #fde047;
  font-size: 0.95rem;
}

.final-round-banner__icon {
  font-size: 1.2rem;
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
