/**
 * Estado del juego Farkle (partida en curso).
 * Gestiona jugadores, dados, puntuación, turnos y transiciones.
 *
 * @param {import('vue').Ref<number>} myPlayerIndex - Índice del jugador actual (0 o 1)
 */
import { ref } from 'vue';
import {
  DEFAULT_DICE_COUNT,
  DICE_SIDES,
  INITIAL_PLAYERS,
  ROLL_ANIMATION_INTERVAL_MS,
  ROLL_DISPLAY_MS,
  TOAST_DELAY_MS,
  TOAST_DURATION_MS,
} from '@/config.js';
import { game as msg } from '@/messages.js';

export function useGameState(myPlayerIndex) {
  const players = ref(INITIAL_PLAYERS.map((p) => ({ ...p })));
  const currentPlayerIndex = ref(0);
  const victoryScore = ref(2000);
  const winnerIndex = ref(null);
  const finalRoundTriggerIndex = ref(null);
  const finishedByDisconnect = ref(false);

  const dices = ref([]);
  const remainingDiceCount = ref(DEFAULT_DICE_COUNT);
  const isRolling = ref(false);
  const selected = ref([]);
  const hasApartadoThisRoll = ref(false);
  const hasRolledThisTurn = ref(false);
  const isTurnEnding = ref(false);

  const turnPoints = ref(0);
  const turnMoves = ref([]);
  const statusMessage = ref('');
  const statusKind = ref('info');

  const farklePendingTransition = ref(false);
  const pendingGameState = ref(null);
  const rollResultPending = ref(false);

  const rollResultTime = ref(null);
  const timerIds = {
    rollAnimationTimeoutId: null,
    rollPlaceholderIntervalId: null,
    farkleDelayTimeoutId: null,
    farkleToastTimeoutId: null,
  };

  function applyGameState(data, options = {}) {
    const { preserveDice = false } = options;
    const ps = data.players || [];
    players.value = ps.map((p) => ({
      name: p.name || msg.playerDefault,
      total: p.total ?? 0,
      // Si el backend envía "active", úsalo; por defecto asumimos true
      active: p.active ?? true,
    }));

    currentPlayerIndex.value = data.currentPlayerIndex ?? 0;
    victoryScore.value = data.victoryScore ?? 2000;
    winnerIndex.value = data.winnerIndex >= 0 ? data.winnerIndex : null;
    finalRoundTriggerIndex.value = data.finalRoundTriggerIndex >= 0 ? data.finalRoundTriggerIndex : null;

    const diceArr = data.dice || [];
    if (!preserveDice) {
      dices.value = diceArr.map((d) => ({ value: d.value ?? 1, held: !!d.held }));
      const selIndices = data.selectedIndices || [];
      selected.value = dices.value.map((_, i) => selIndices.includes(i));
    }

    let rem = data.remainingDiceCount;
    if (rem === undefined || rem === null) {
      rem = diceArr.length === 0 ? DEFAULT_DICE_COUNT : diceArr.filter((d) => !d?.held).length;
    } else if (diceArr.length === 0 && rem === 0) {
      rem = DEFAULT_DICE_COUNT;
    }
    remainingDiceCount.value = rem;
    turnPoints.value = data.turnPoints ?? 0;

    const moves = data.turnMoves || [];
    turnMoves.value = moves.map((m) => ({
      id: m.id ?? 0,
      values: m.values || [],
      points: m.points ?? 0,
    }));

    if (!rollResultPending.value) isRolling.value = false;
    isTurnEnding.value = false;
    hasRolledThisTurn.value = preserveDice ? false : dices.value.length > 0;
    if (statusKind.value === 'error') {
      statusMessage.value = '';
      statusKind.value = 'info';
    }
    if (preserveDice) {
      const newPlayerIdx = data.currentPlayerIndex ?? 0;
      const isNewPlayerTurn = myPlayerIndex.value === newPlayerIdx;
      statusMessage.value = isNewPlayerTurn
        ? msg.diceReady(remainingDiceCount.value)
        : msg.waitingForTurn(ps[newPlayerIdx]?.name || msg.playerFallback);
      statusKind.value = 'info';
    }
    hasApartadoThisRoll.value =
      dices.value.some((d) => d.held) || (dices.value.length === 0 && turnPoints.value > 0);
  }

  function clearRollingTimers() {
    if (timerIds.rollAnimationTimeoutId) {
      clearTimeout(timerIds.rollAnimationTimeoutId);
      timerIds.rollAnimationTimeoutId = null;
    }
    if (timerIds.rollPlaceholderIntervalId) {
      clearInterval(timerIds.rollPlaceholderIntervalId);
      timerIds.rollPlaceholderIntervalId = null;
    }
    if (timerIds.farkleDelayTimeoutId) {
      clearTimeout(timerIds.farkleDelayTimeoutId);
      timerIds.farkleDelayTimeoutId = null;
    }
    if (timerIds.farkleToastTimeoutId) {
      clearTimeout(timerIds.farkleToastTimeoutId);
      timerIds.farkleToastTimeoutId = null;
    }
  }

  function resetGame() {
    clearRollingTimers();
    isRolling.value = false;
    players.value = INITIAL_PLAYERS.map((p) => ({ ...p }));
    currentPlayerIndex.value = 0;
    winnerIndex.value = null;
    turnPoints.value = 0;
    turnMoves.value = [];
    remainingDiceCount.value = DEFAULT_DICE_COUNT;
    dices.value = [];
    selected.value = [];
    hasApartadoThisRoll.value = false;
    hasRolledThisTurn.value = false;
    finalRoundTriggerIndex.value = null;
    finishedByDisconnect.value = false;
    isTurnEnding.value = false;
    statusMessage.value = '';
    statusKind.value = 'info';
  }

  return {
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
    TOAST_DELAY_MS,
    TOAST_DURATION_MS,
    DICE_SIDES,
    ROLL_ANIMATION_INTERVAL_MS,
  };
}
