package main

import "fmt"

// Constantes de puntuación Farkle
const (
	pointsSingle1  = 100
	pointsSingle5  = 50
	pointsTriple1  = 1000
	pointsTriple2  = 200
	pointsTriple3  = 300
	pointsTriple4  = 400
	pointsTriple5  = 500
	pointsTriple6  = 600
	pointsFour     = 1000
	pointsFive     = 2000
	pointsSix      = 3000
	pointsStraight = 1500
	pointsThreePair = 1500
	pointsFourPair = 1500
	pointsTwoTriple = 2500
)

const (
	diceMin = 1
	diceMax = 6
)

// emptyCounts devuelve un mapa de conteos vacío para valores 1-6.
func emptyCounts() map[int]int {
	return map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0}
}

// makeCounts cuenta cuántos dados de cada valor (1-6) hay.
func makeCounts(values []int) map[int]int {
	counts := emptyCounts()
	for _, v := range values {
		if v >= diceMin && v <= diceMax {
			counts[v]++
		}
	}
	return counts
}

func cloneCounts(counts map[int]int) map[int]int {
	c := make(map[int]int)
	for k, v := range counts {
		c[k] = v
	}
	return c
}

func subtractCounts(counts, use map[int]int) map[int]int {
	next := cloneCounts(counts)
	for v := diceMin; v <= diceMax; v++ {
		next[v] -= use[v]
		if next[v] < 0 {
			next[v] = 0
		}
	}
	return next
}

func allZero(counts map[int]int) bool {
	for v := diceMin; v <= diceMax; v++ {
		if counts[v] != 0 {
			return false
		}
	}
	return true
}

type combo struct {
	use    map[int]int
	points int
}

func possibleCombos(counts map[int]int) []combo {
	var combos []combo
	total := 0
	for v := diceMin; v <= diceMax; v++ {
		total += counts[v]
	}

	// Escalera 1-6
	if counts[1] >= 1 && counts[2] >= 1 && counts[3] >= 1 && counts[4] >= 1 && counts[5] >= 1 && counts[6] >= 1 {
		combos = append(combos, combo{map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1}, pointsStraight})
	}

	if total == Cfg.NumDice {
		pairCount := 0
		other := false
		for v := diceMin; v <= diceMax; v++ {
			if counts[v] == 2 {
				pairCount++
			} else if counts[v] != 0 {
				other = true
			}
		}
		if pairCount == 3 && !other {
			combos = append(combos, combo{cloneCounts(counts), pointsThreePair})
		}

		for v := diceMin; v <= diceMax; v++ {
			if counts[v] == 4 {
				for w := diceMin; w <= diceMax; w++ {
					if w != v && counts[w] == 2 {
						combos = append(combos, combo{cloneCounts(counts), pointsFourPair})
						break
					}
				}
			}
		}

		for v := diceMin; v <= diceMax; v++ {
			if counts[v] == 3 {
				for w := v + 1; w <= diceMax; w++ {
					if counts[w] == 3 {
						combos = append(combos, combo{cloneCounts(counts), pointsTwoTriple})
						break
					}
				}
			}
		}
	}

	tripleScores := map[int]int{1: pointsTriple1, 2: pointsTriple2, 3: pointsTriple3, 4: pointsTriple4, 5: pointsTriple5, 6: pointsTriple6}
	for v := diceMin; v <= diceMax; v++ {
		if counts[v] >= 3 {
			use := emptyCounts()
			use[v] = 3
			combos = append(combos, combo{use, tripleScores[v]})
		}
		if counts[v] >= 4 {
			use := emptyCounts()
			use[v] = 4
			combos = append(combos, combo{use, pointsFour})
		}
		if counts[v] >= 5 {
			use := emptyCounts()
			use[v] = 5
			combos = append(combos, combo{use, pointsFive})
		}
		if counts[v] >= 6 {
			use := emptyCounts()
			use[v] = 6
			combos = append(combos, combo{use, pointsSix})
		}
	}

	if counts[1] >= 1 {
		use := emptyCounts()
		use[1] = 1
		combos = append(combos, combo{use, pointsSingle1})
	}
	if counts[5] >= 1 {
		use := emptyCounts()
		use[5] = 1
		combos = append(combos, combo{use, pointsSingle5})
	}

	return combos
}

func countsKey(counts map[int]int) string {
	return fmt.Sprintf("%d,%d,%d,%d,%d,%d",
		counts[1], counts[2], counts[3], counts[4], counts[5], counts[6])
}

type scoreResult struct {
	valid  bool
	points int
}

func bestScoreUsingAll(counts map[int]int, memo map[string]scoreResult) (valid bool, points int) {
	key := countsKey(counts)
	if c, ok := memo[key]; ok {
		return c.valid, c.points
	}

	if allZero(counts) {
		memo[key] = scoreResult{true, 0}
		return true, 0
	}

	bestValid := false
	bestPoints := 0
	combos := possibleCombos(counts)

	for _, combo := range combos {
		ok := true
		for v := diceMin; v <= diceMax; v++ {
			if combo.use[v] > counts[v] {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		next := subtractCounts(counts, combo.use)
		subValid, subPoints := bestScoreUsingAll(next, memo)
		if !subValid {
			continue
		}
		total := combo.points + subPoints
		if !bestValid || total > bestPoints {
			bestValid = true
			bestPoints = total
		}
	}

	memo[key] = scoreResult{bestValid, bestPoints}
	return bestValid, bestPoints
}

// ScoreSelection valida la selección de dados y devuelve si es válida y los puntos.
func ScoreSelection(values []int) (valid bool, points int) {
	if len(values) == 0 {
		return false, 0
	}
	counts := makeCounts(values)
	memo := make(map[string]scoreResult)
	valid, points = bestScoreUsingAll(counts, memo)
	if !valid || points <= 0 {
		return false, 0
	}
	return true, points
}

// HasAnyScoringOption indica si hay alguna combinación puntuable en los dados.
func HasAnyScoringOption(values []int) bool {
	counts := makeCounts(values)
	if counts[1] > 0 || counts[5] > 0 {
		return true
	}
	for v := diceMin; v <= diceMax; v++ {
		if counts[v] >= 3 {
			return true
		}
	}
	total := 0
	for v := diceMin; v <= diceMax; v++ {
		total += counts[v]
	}
	if total != Cfg.NumDice {
		return false
	}
	if counts[1] == 1 && counts[2] == 1 && counts[3] == 1 && counts[4] == 1 && counts[5] == 1 && counts[6] == 1 {
		return true
	}
	pairCount := 0
	for v := diceMin; v <= diceMax; v++ {
		if counts[v] == 2 {
			pairCount++
		}
	}
	if pairCount == 3 {
		return true
	}
	for v := diceMin; v <= diceMax; v++ {
		if counts[v] == 4 {
			for w := diceMin; w <= diceMax; w++ {
				if w != v && counts[w] == 2 {
					return true
				}
			}
		}
	}
	for v := diceMin; v <= diceMax; v++ {
		if counts[v] == 3 {
			for w := v + 1; w <= diceMax; w++ {
				if counts[w] == 3 {
					return true
				}
			}
		}
	}
	return false
}
