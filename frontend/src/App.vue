<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import Die from '@/components/Die.vue';
import LobbyModal from '@/components/LobbyModal.vue';
import { useWebSocket } from '@/composables/useWebSocket.js';
import { hasAnyScoringOption, scoreSelection } from '@/game/farkleScoring.js';

const ws = useWebSocket();
const inGame = ref(false);
const gameCode = ref('');
const myPlayerIndex = ref(-1);

onMounted(() => {
  ws.connect();
});

const onLobbySuccess = ({ gameCode: code, playerIndex }) => {
  gameCode.value = code;
  myPlayerIndex.value = playerIndex;
  inGame.value = true;
};

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

const ROLL_DURATION_MS = 1500;
const END_TURN_DELAY_MS = 2000;

let rollIntervalId = null;
let rollTimeoutId = null;

const scheduleNextPlayer = () => {
  if (winnerIndex.value !== null) return;
  isTurnEnding.value = true;
  const finishedIndex = currentPlayerIndex.value;
  setTimeout(() => {
    if (winnerIndex.value !== null) return;

    // Si ya se ha disparado la ronda final y acaba de jugar el otro jugador,
    // aquí termina la partida comparando puntuaciones.
    if (
      finalRoundTriggerIndex.value !== null
      && finishedIndex !== finalRoundTriggerIndex.value
    ) {
      isTurnEnding.value = false;
      const [p0, p1] = players.value;
      const totals = [p0.total, p1.total];
      let winner = 0;
      if (totals[1] > totals[0]) winner = 1;

      winnerIndex.value = winner;
      statusKind.value = 'success';
      if (totals[0] === totals[1]) {
        statusMessage.value = `Empate a ${totals[0]} puntos. Gana ${players.value[winner].name} por desempate.`;
      } else {
        statusMessage.value = `${players.value[winner].name} gana con ${totals[winner]} puntos.`;
      }
      return;
    }

    nextPlayer();
  }, END_TURN_DELAY_MS);
};

const rollOnce = () => {
  if (dices.value.length === 0) {
    dices.value = Array.from({ length: remainingDiceCount.value }, () => ({
      value: Math.floor(Math.random() * 6) + 1,
      held: false,
    }));
    selected.value = dices.value.map(() => false);
    return;
  }

  dices.value = dices.value.map((die) =>
    die.held ? die : { ...die, value: Math.floor(Math.random() * 6) + 1 },
  );
  selected.value = dices.value.map(() => false);
};

const clearRollingTimers = () => {
  if (rollIntervalId !== null) {
    clearInterval(rollIntervalId);
    rollIntervalId = null;
  }
  if (rollTimeoutId !== null) {
    clearTimeout(rollTimeoutId);
    rollTimeoutId = null;
  }
};

const rollDices = () => {
  if (isTurnEnding.value) return;
  if (isRolling.value) return;
  if (winnerIndex.value !== null) return;
  // Regla: tras cada tirada debes apartar al menos una combinación puntuable
  if (dices.value.length > 0 && !hasApartadoThisRoll.value) return;

  isRolling.value = true;
  statusMessage.value = '';
  hasApartadoThisRoll.value = false;
  clearRollingTimers();

  rollOnce(); // primera tirada inmediata
  rollIntervalId = setInterval(rollOnce, 80);

  rollTimeoutId = setTimeout(() => {
    clearRollingTimers();
    rollOnce(); // tirada final
    isRolling.value = false;
    hasApartadoThisRoll.value = false;
    hasRolledThisTurn.value = true;

    const activeValues = dices.value.filter((d) => !d.held).map((d) => d.value);
    if (!hasAnyScoringOption(activeValues)) {
      statusKind.value = 'warn';
      statusMessage.value = `Farkle: pierdes ${turnPoints.value} puntos del turno.`;
      turnPoints.value = 0;
      turnMoves.value = [];
      selected.value = dices.value.map(() => false);
      hasApartadoThisRoll.value = false;
      scheduleNextPlayer();
    }
  }, ROLL_DURATION_MS);
};

const toggleSelect = (index) => {
  if (isTurnEnding.value) return;
  if (isRolling.value) return;
  if (winnerIndex.value !== null) return;
  if (!hasRolledThisTurn.value) return;
  if (!dices.value.length) return;
  if (dices.value[index]?.held) return;
  selected.value[index] = !selected.value[index];
};

const hasSelection = computed(() => selected.value.some((v) => v));

const apartarSeleccionados = () => {
  if (!hasSelection.value) return;
  if (isRolling.value) return;
  if (winnerIndex.value !== null) return;
  if (!hasRolledThisTurn.value) return;
  if (!dices.value.length) return;

  const currentDices = dices.value;
  const currentSelected = selected.value;

  const pickedValues = currentDices
    .filter((die, idx) => currentSelected[idx] && !die.held)
    .map((die) => die.value);
  const scoring = scoreSelection(pickedValues);

  if (!scoring.valid) {
    statusKind.value = 'error';
    statusMessage.value = 'Selección inválida: todos los dados seleccionados deben puntuar.';
    return;
  }

  statusMessage.value = '';
  turnPoints.value += scoring.points;
  hasApartadoThisRoll.value = true;
  turnMoves.value.push({
    id: turnMoves.value.length + 1,
    values: pickedValues,
    points: scoring.points,
    breakdown: scoring.breakdown,
  });

  // Mantener todos los dados visibles; los apartados se quedan "held" (desactivados)
  dices.value = currentDices.map((die, idx) =>
    currentSelected[idx] && !die.held ? { ...die, held: true } : die,
  );
  remainingDiceCount.value = dices.value.filter((d) => !d.held).length;
  selected.value = dices.value.map(() => false);

  // Mano limpia: se puntúa usando los 6 dados -> vuelves a tener 6 para tirar
  if (remainingDiceCount.value === 0) {
    remainingDiceCount.value = 6;
    dices.value = [];
    selected.value = [];
    hasApartadoThisRoll.value = true;
    statusKind.value = 'success';
    statusMessage.value = 'Mano limpia: puedes volver a tirar los 6 dados.';
  }
};

// Regla: no se puede "plantarse" con una tirada sin haber apartado al menos una vez
const canBank = computed(() =>
  turnPoints.value > 0
  && !isRolling.value
  && winnerIndex.value === null
  && (dices.value.length === 0 || hasApartadoThisRoll.value),
);

const bankTurn = () => {
  if (isTurnEnding.value) return;
  if (!canBank.value) return;

  const p = players.value[currentPlayerIndex.value];
  p.total += turnPoints.value;
  p.rounds.push({
    id: p.rounds.length + 1,
    points: turnPoints.value,
    moves: turnMoves.value,
  });

  // Si aún no se ha disparado la ronda final y este jugador alcanza o supera la meta,
  // se marca el inicio de la ronda final: el otro jugador tendrá un último turno.
  if (finalRoundTriggerIndex.value === null && p.total >= victoryScore.value) {
    finalRoundTriggerIndex.value = currentPlayerIndex.value;
    statusKind.value = 'success';
    statusMessage.value = `${p.name} alcanza ${p.total} puntos. Turno final para el otro jugador.`;
    scheduleNextPlayer();
    return;
  }

  scheduleNextPlayer();
};

const nextPlayer = () => {
  clearRollingTimers();
  isRolling.value = false;

  turnPoints.value = 0;
  turnMoves.value = [];
  isTurnEnding.value = false;
  hasRolledThisTurn.value = false;

  if (dices.value.length > 0) {
    // Reactivar todos los dados manteniendo el último valor visible
    dices.value = dices.value.map((die) => ({
      value: die.value,
      held: false,
    }));
    remainingDiceCount.value = dices.value.length;
    selected.value = dices.value.map(() => false);
    // Para el nuevo jugador, estos dados cuentan como "listos para tirar" sin exigir un apartado previo
    hasApartadoThisRoll.value = true;
  } else {
    remainingDiceCount.value = 6;
    selected.value = [];
    hasApartadoThisRoll.value = false;
  }

  currentPlayerIndex.value = (currentPlayerIndex.value + 1) % players.value.length;
  statusMessage.value = '';
  statusKind.value = 'info';
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
      type="button"
      class="btn btn--secondary"
      @click="resetGame"
    >
      Nueva partida
    </button>
    
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
            :disabled="winnerIndex !== null || die.held || !hasRolledThisTurn"
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
        :disabled="isRolling || winnerIndex !== null || (dices.length > 0 && !hasApartadoThisRoll)"
        @click="rollDices"
      >
        {{ isRolling ? 'Tirando...' : 'Tirar dados' }}
      </button>

      <button
        type="button"
        class="btn btn--secondary"
        :disabled="isRolling || !hasSelection || winnerIndex !== null || !hasRolledThisTurn"
        @click="apartarSeleccionados"
      >
        Apartar
      </button>

      <button
        type="button"
        class="btn btn--bank"
        :disabled="!canBank"
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
