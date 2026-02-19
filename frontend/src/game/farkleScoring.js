function countsKey(counts) {
  // counts: 1..6
  return [counts[1], counts[2], counts[3], counts[4], counts[5], counts[6]].join(',');
}

function cloneCounts(counts) {
  return {
    1: counts[1],
    2: counts[2],
    3: counts[3],
    4: counts[4],
    5: counts[5],
    6: counts[6],
  };
}

function subtractCounts(counts, use) {
  const next = cloneCounts(counts);
  for (let v = 1; v <= 6; v += 1) {
    next[v] -= use[v] ?? 0;
  }
  return next;
}

function allZero(counts) {
  for (let v = 1; v <= 6; v += 1) {
    if (counts[v] !== 0) return false;
  }
  return true;
}

function makeCounts(values) {
  const counts = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
  for (const v of values) counts[v] += 1;
  return counts;
}

function comboStraight(label, required, points) {
  const use = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
  for (const v of required) use[v] = 1;
  return { label, use, points };
}

function comboSingle(label, v, points) {
  const use = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
  use[v] = 1;
  return { label, use, points };
}

function possibleCombos(counts) {
  const combos = [];

  const total =
    counts[1] + counts[2] + counts[3] + counts[4] + counts[5] + counts[6];

  // Escalera 1-6
  if ([1, 2, 3, 4, 5, 6].every((v) => counts[v] >= 1)) {
    combos.push(comboStraight('Escalera 1-6', [1, 2, 3, 4, 5, 6], 1500));
  }

  // Tres parejas (exactamente 3 valores con 2 dados cada uno)
  if (total === 6) {
    const pairValues = [];
    let other = false;
    for (let v = 1; v <= 6; v += 1) {
      if (counts[v] === 2) pairValues.push(v);
      else if (counts[v] !== 0) other = true;
    }
    if (pairValues.length === 3 && !other) {
      combos.push({
        label: 'Tres parejas',
        use: cloneCounts(counts),
        points: 1500,
      });
    }

    // Cuatro iguales y una pareja (4 + 2)
    for (let v = 1; v <= 6; v += 1) {
      if (counts[v] === 4) {
        for (let w = 1; w <= 6; w += 1) {
          if (w !== v && counts[w] === 2) {
            combos.push({
              label: 'Cuatro iguales y una pareja',
              use: cloneCounts(counts),
              points: 1500,
            });
          }
        }
      }
    }

    // Dos tríos (3 + 3)
    for (let v = 1; v <= 6; v += 1) {
      if (counts[v] === 3) {
        for (let w = v + 1; w <= 6; w += 1) {
          if (counts[w] === 3) {
            combos.push({
              label: 'Dos tríos',
              use: cloneCounts(counts),
              points: 2500,
            });
          }
        }
      }
    }
  }

  // Tríos específicos
  for (let v = 1; v <= 6; v += 1) {
    if (counts[v] >= 3) {
      const use = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
      use[v] = 3;
      const tripleScores = {
        1: 1000,
        2: 200,
        3: 300,
        4: 400,
        5: 500,
        6: 600,
      };
      combos.push({
        label: `Trío de ${v}`,
        use,
        points: tripleScores[v],
      });
    }
  }

  // 4, 5, 6 iguales (cualquier número)
  for (let v = 1; v <= 6; v += 1) {
    if (counts[v] >= 4) {
      const use4 = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
      use4[v] = 4;
      combos.push({
        label: `4 de ${v}`,
        use: use4,
        points: 1000,
      });
    }
    if (counts[v] >= 5) {
      const use5 = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
      use5[v] = 5;
      combos.push({
        label: `5 de ${v}`,
        use: use5,
        points: 2000,
      });
    }
    if (counts[v] >= 6) {
      const use6 = { 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0 };
      use6[v] = 6;
      combos.push({
        label: `6 de ${v}`,
        use: use6,
        points: 3000,
      });
    }
  }

  // Singles
  if (counts[1] >= 1) combos.push(comboSingle('1 suelto', 1, 100));
  if (counts[5] >= 1) combos.push(comboSingle('5 suelto', 5, 50));

  return combos;
}

function bestScoreUsingAll(counts) {
  const memo = new Map();

  /** @returns {{ valid: boolean, points: number, breakdown: Array<{label: string, points: number, use: object}> }} */
  function rec(state) {
    const key = countsKey(state);
    const cached = memo.get(key);
    if (cached) return cached;

    if (allZero(state)) {
      const res = { valid: true, points: 0, breakdown: [] };
      memo.set(key, res);
      return res;
    }

    let best = { valid: false, points: 0, breakdown: [] };
    const combos = possibleCombos(state);

    for (const combo of combos) {
      let ok = true;
      for (let v = 1; v <= 6; v += 1) {
        const need = combo.use[v] ?? 0;
        if (need > state[v]) {
          ok = false;
          break;
        }
      }
      if (!ok) continue;

      const next = subtractCounts(state, combo.use);
      const sub = rec(next);
      if (!sub.valid) continue;

      const total = combo.points + sub.points;
      if (!best.valid || total > best.points) {
        best = {
          valid: true,
          points: total,
          breakdown: [{ label: combo.label, points: combo.points, use: combo.use }, ...sub.breakdown],
        };
      }
    }

    memo.set(key, best);
    return best;
  }

  return rec(counts);
}

export function scoreSelection(values) {
  if (!values || values.length === 0) return { valid: false, points: 0, breakdown: [] };
  const counts = makeCounts(values);
  const best = bestScoreUsingAll(counts);
  if (!best.valid || best.points <= 0) return { valid: false, points: 0, breakdown: [] };
  return best;
}

export function hasAnyScoringOption(rollValues) {
  const counts = makeCounts(rollValues);
  const total =
    counts[1] + counts[2] + counts[3] + counts[4] + counts[5] + counts[6];

  // 1 sueltos y 5 sueltos
  if (counts[1] > 0 || counts[5] > 0) return true;

  // Tríos o más de cualquier número
  for (let v = 1; v <= 6; v += 1) {
    if (counts[v] >= 3) return true;
  }

  if (total === 6) {
    // Escalera 1-6
    if ([1, 2, 3, 4, 5, 6].every((v) => counts[v] === 1)) return true;

    // Tres parejas
    const pairValues = [];
    let other = false;
    for (let v = 1; v <= 6; v += 1) {
      if (counts[v] === 2) pairValues.push(v);
      else if (counts[v] !== 0) other = true;
    }
    if (pairValues.length === 3 && !other) return true;

    // Cuatro iguales y una pareja
    for (let v = 1; v <= 6; v += 1) {
      if (counts[v] === 4) {
        for (let w = 1; w <= 6; w += 1) {
          if (w !== v && counts[w] === 2) return true;
        }
      }
    }

    // Dos tríos
    for (let v = 1; v <= 6; v += 1) {
      if (counts[v] === 3) {
        for (let w = v + 1; w <= 6; w += 1) {
          if (counts[w] === 3) return true;
        }
      }
    }
  }

  return false;
}

