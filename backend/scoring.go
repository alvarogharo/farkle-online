package main

import "fmt"

// makeCounts cuenta cuántos dados de cada valor (1-6) hay.
func makeCounts(values []int) map[int]int {
	counts := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0}
	for _, v := range values {
		if v >= 1 && v <= 6 {
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
	for v := 1; v <= 6; v++ {
		next[v] -= use[v]
		if next[v] < 0 {
			next[v] = 0
		}
	}
	return next
}

func allZero(counts map[int]int) bool {
	for v := 1; v <= 6; v++ {
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
	for v := 1; v <= 6; v++ {
		total += counts[v]
	}

	// Escalera 1-6
	if counts[1] >= 1 && counts[2] >= 1 && counts[3] >= 1 && counts[4] >= 1 && counts[5] >= 1 && counts[6] >= 1 {
		combos = append(combos, combo{map[int]int{1: 1, 2: 1, 3: 1, 4: 1, 5: 1, 6: 1}, 1500})
	}

	if total == 6 {
		pairCount := 0
		other := false
		for v := 1; v <= 6; v++ {
			if counts[v] == 2 {
				pairCount++
			} else if counts[v] != 0 {
				other = true
			}
		}
		if pairCount == 3 && !other {
			combos = append(combos, combo{cloneCounts(counts), 1500})
		}

		for v := 1; v <= 6; v++ {
			if counts[v] == 4 {
				for w := 1; w <= 6; w++ {
					if w != v && counts[w] == 2 {
						combos = append(combos, combo{cloneCounts(counts), 1500})
						break
					}
				}
			}
		}

		for v := 1; v <= 6; v++ {
			if counts[v] == 3 {
				for w := v + 1; w <= 6; w++ {
					if counts[w] == 3 {
						combos = append(combos, combo{cloneCounts(counts), 2500})
						break
					}
				}
			}
		}
	}

	tripleScores := map[int]int{1: 1000, 2: 200, 3: 300, 4: 400, 5: 500, 6: 600}
	for v := 1; v <= 6; v++ {
		if counts[v] >= 3 {
			use := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0}
			use[v] = 3
			combos = append(combos, combo{use, tripleScores[v]})
		}
		if counts[v] >= 4 {
			use := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0}
			use[v] = 4
			combos = append(combos, combo{use, 1000})
		}
		if counts[v] >= 5 {
			use := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0}
			use[v] = 5
			combos = append(combos, combo{use, 2000})
		}
		if counts[v] >= 6 {
			use := map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0}
			use[v] = 6
			combos = append(combos, combo{use, 3000})
		}
	}

	if counts[1] >= 1 {
		combos = append(combos, combo{map[int]int{1: 1, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0}, 100})
	}
	if counts[5] >= 1 {
		combos = append(combos, combo{map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 1, 6: 0}, 50})
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
		for v := 1; v <= 6; v++ {
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
	for v := 1; v <= 6; v++ {
		if counts[v] >= 3 {
			return true
		}
	}
	total := 0
	for v := 1; v <= 6; v++ {
		total += counts[v]
	}
	if total != 6 {
		return false
	}
	if counts[1] == 1 && counts[2] == 1 && counts[3] == 1 && counts[4] == 1 && counts[5] == 1 && counts[6] == 1 {
		return true
	}
	pairCount := 0
	for v := 1; v <= 6; v++ {
		if counts[v] == 2 {
			pairCount++
		}
	}
	if pairCount == 3 {
		return true
	}
	for v := 1; v <= 6; v++ {
		if counts[v] == 4 {
			for w := 1; w <= 6; w++ {
				if w != v && counts[w] == 2 {
					return true
				}
			}
		}
	}
	for v := 1; v <= 6; v++ {
		if counts[v] == 3 {
			for w := v + 1; w <= 6; w++ {
				if counts[w] == 3 {
					return true
				}
			}
		}
	}
	return false
}
