// simulate_hotdice.js
// Simula turnos de Farkle usando las mismas reglas de scoring que el backend Go,
// y elige siempre la selección válida que usa más dados posibles para maximizar
// la probabilidad de llegar al 3er Hot Dice sin Farkle.

// --- Config básicos del juego ---
const NUM_DICE = 6;
const diceMin = 1;
const diceMax = 6;

// Puntuaciones como en backend/scoring.go
const pointsSingle1 = 100;
const pointsSingle5 = 50;
const pointsTriple = {
  1: 1000,
  2: 200,
  3: 300,
  4: 400,
  5: 500,
  6: 600,
};
const pointsFour = 1000;
const pointsFive = 2000;
const pointsSix = 3000;
const pointsStraight = 1500;
const pointsThreePair = 1500;
const pointsFourPair = 1500;
const pointsTwoTriple = 2500;

// ------- Utilidades de conteos (como scoring.go) -------

function emptyCounts() {
  return { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
}

function makeCounts(values) {
  const counts = emptyCounts();
  for (const v of values) {
    if (v >= diceMin && v <= diceMax) counts[v]++;
  }
  return counts;
}

function cloneCounts(counts) {
  const c = {};
  for (const k in counts) c[k] = counts[k];
  return c;
}

function subtractCounts(counts, use) {
  const next = cloneCounts(counts);
  for (let v = diceMin; v <= diceMax; v++) {
    next[v] -= use[v] || 0;
    if (next[v] < 0) next[v] = 0;
  }
  return next;
}

function allZero(counts) {
  for (let v = diceMin; v <= diceMax; v++) {
    if (counts[v] !== 0) return false;
  }
  return true;
}

// ------- Generación de combinaciones puntuables -------

function possibleCombos(counts) {
  const combos = [];
  let total = 0;
  for (let v = diceMin; v <= diceMax; v++) {
    total += counts[v];
  }

  // Escalera 1-6
  if (
    counts[1] >= 1 &&
    counts[2] >= 1 &&
    counts[3] >= 1 &&
    counts[4] >= 1 &&
    counts[5] >= 1 &&
    counts[6] >= 1
  ) {
    combos.push({
      use: { 1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1 },
      points: pointsStraight,
    });
  }

  // Combinaciones que requieren usar exactamente NUM_DICE (tres parejas, etc.)
  if (total === NUM_DICE) {
    // Tres parejas
    let pairCount = 0;
    let other = false;
    for (let v = diceMin; v <= diceMax; v++) {
      if (counts[v] === 2) pairCount++;
      else if (counts[v] !== 0) other = true;
    }
    if (pairCount === 3 && !other) {
      combos.push({ use: cloneCounts(counts), points: pointsThreePair });
    }

    // Cuatro de un valor + pareja de otro
    for (let v = diceMin; v <= diceMax; v++) {
      if (counts[v] === 4) {
        for (let w = diceMin; w <= diceMax; w++) {
          if (w !== v && counts[w] === 2) {
            combos.push({ use: cloneCounts(counts), points: pointsFourPair });
            break;
          }
        }
      }
    }

    // Dos tríos
    for (let v = diceMin; v <= diceMax; v++) {
      if (counts[v] === 3) {
        for (let w = v + 1; w <= diceMax; w++) {
          if (counts[w] === 3) {
            combos.push({ use: cloneCounts(counts), points: pointsTwoTriple });
            break;
          }
        }
      }
    }
  }

  // Triples, cuádruples, quíntuples, séxtuples
  for (let v = diceMin; v <= diceMax; v++) {
    if (counts[v] >= 3) {
      const use = emptyCounts();
      use[v] = 3;
      combos.push({ use, points: pointsTriple[v] });
    }
    if (counts[v] >= 4) {
      const use = emptyCounts();
      use[v] = 4;
      combos.push({ use, points: pointsFour });
    }
    if (counts[v] >= 5) {
      const use = emptyCounts();
      use[v] = 5;
      combos.push({ use, points: pointsFive });
    }
    if (counts[v] >= 6) {
      const use = emptyCounts();
      use[v] = 6;
      combos.push({ use, points: pointsSix });
    }
  }

  // Dados sueltos 1 y 5
  if (counts[1] >= 1) {
    const use = emptyCounts();
    use[1] = 1;
    combos.push({ use, points: pointsSingle1 });
  }
  if (counts[5] >= 1) {
    const use = emptyCounts();
    use[5] = 1;
    combos.push({ use, points: pointsSingle5 });
  }

  return combos;
}

function countsKey(counts) {
  return `${counts[1]},${counts[2]},${counts[3]},${counts[4]},${counts[5]},${counts[6]}`;
}

const globalMemo = new Map(); // cache compartida entre ScoreSelection

function bestScoreUsingAll(counts) {
  const key = countsKey(counts);
  if (globalMemo.has(key)) return globalMemo.get(key);

  if (allZero(counts)) {
    const res = { valid: true, points: 0 };
    globalMemo.set(key, res);
    return res;
  }

  let bestValid = false;
  let bestPoints = 0;
  const combos = possibleCombos(counts);

  for (const combo of combos) {
    let ok = true;
    for (let v = diceMin; v <= diceMax; v++) {
      if ((combo.use[v] || 0) > counts[v]) {
        ok = false;
        break;
      }
    }
    if (!ok) continue;

    const next = subtractCounts(counts, combo.use);
    const sub = bestScoreUsingAll(next);
    if (!sub.valid) continue;

    const total = combo.points + sub.points;
    if (!bestValid || total > bestPoints) {
      bestValid = true;
      bestPoints = total;
    }
  }

  const res = { valid: bestValid, points: bestPoints };
  globalMemo.set(key, res);
  return res;
}

// Valida que TODOS los dados de la selección puntúan.
function scoreSelection(values) {
  if (values.length === 0) return { valid: false, points: 0 };
  const counts = makeCounts(values);
  const { valid, points } = bestScoreUsingAll(counts);
  if (!valid || points <= 0) return { valid: false, points: 0 };
  return { valid: true, points };
}

// ¿Hay alguna opción puntuable en estos dados?
function hasAnyScoringOption(values) {
  const counts = makeCounts(values);
  if (counts[1] > 0 || counts[5] > 0) return true;
  for (let v = diceMin; v <= diceMax; v++) {
    if (counts[v] >= 3) return true;
  }
  let total = 0;
  for (let v = diceMin; v <= diceMax; v++) total += counts[v];
  if (total !== NUM_DICE) return false;

  // Escalera 1-6
  if (
    counts[1] === 1 &&
    counts[2] === 1 &&
    counts[3] === 1 &&
    counts[4] === 1 &&
    counts[5] === 1 &&
    counts[6] === 1
  ) {
    return true;
  }

  // Tres parejas
  let pairCount = 0;
  for (let v = diceMin; v <= diceMax; v++) {
    if (counts[v] === 2) pairCount++;
  }
  if (pairCount === 3) return true;

  return false;
}

// ------- Estrategia: elegir la selección válida que usa MÁS dados -------

function chooseGreedySelection(values) {
  const n = values.length;
  let bestMask = 0;
  let bestLen = 0;
  let bestPoints = 0;

  // Probar todos los subconjuntos no vacíos (hasta 6 dados: 2^6-1 = 63)
  const maxMask = 1 << n;
  for (let mask = 1; mask < maxMask; mask++) {
    const subset = [];
    for (let i = 0; i < n; i++) {
      if (mask & (1 << i)) subset.push(values[i]);
    }
    const { valid, points } = scoreSelection(subset);
    if (!valid) continue;

    const len = subset.length;
    if (
      len > bestLen ||
      (len === bestLen && points > bestPoints)
    ) {
      bestLen = len;
      bestPoints = points;
      bestMask = mask;
    }
  }

  if (bestMask === 0) return null;

  const indices = [];
  for (let i = 0; i < n; i++) {
    if (bestMask & (1 << i)) indices.push(i);
  }
  return { indices, points: bestPoints };
}

// ------- Simulación de un turno -------

function rollOneDie() {
  return Math.floor(Math.random() * 6) + 1;
}

function rollDice(count) {
  const res = [];
  for (let i = 0; i < count; i++) res.push(rollOneDie());
  return res;
}

// Devuelve true si en este turno se alcanza al menos el 3er Hot Dice sin Farkle.
function simulateTurnToThirdHotDice() {
  let remainingDice = NUM_DICE;
  let hotDiceCount = 0;

  while (true) {
    const values = rollDice(remainingDice);

    if (!hasAnyScoringOption(values)) {
      // Farkle: turno perdido
      return false;
    }

    const sel = chooseGreedySelection(values);
    if (!sel) {
      // En teoría no debería ocurrir si hasAnyScoringOption es true,
      // pero por seguridad lo tratamos como Farkle.
      return false;
    }

    remainingDice -= sel.indices.length;

    if (remainingDice === 0) {
      hotDiceCount++;
      if (hotDiceCount >= 3) {
        return true;
      }
      // Hot dice: se vuelven a tirar los 6
      remainingDice = NUM_DICE;
    }

    // Si aún quedan dados, seguimos el turno.
  }
}

// ------- Main: simular muchos turnos -------

function main() {
  const NUM_TURNS = 1000000;
  let successes = 0;

  console.log(`Iniciando simulación de ${NUM_TURNS} turnos...`);

  const progressStep = Math.max(1, Math.floor(NUM_TURNS / 10));

  for (let i = 0; i < NUM_TURNS; i++) {
    if (simulateTurnToThirdHotDice()) successes++;

    if ((i + 1) % progressStep === 0) {
      const pct = ((i + 1) / NUM_TURNS) * 100;
      console.log(`Progreso: ${pct.toFixed(1)}% (${i + 1}/${NUM_TURNS})`);
    }
  }

  const prob = successes / NUM_TURNS;
  console.log('--- Resultado ---');
  console.log(`Turnos simulados: ${NUM_TURNS}`);
  console.log(`Turnos que llegan a 3er Hot Dice sin Farkle: ${successes}`);
  console.log(`Probabilidad estimada ≈ ${(prob * 100).toFixed(3)}%`);
}

main();