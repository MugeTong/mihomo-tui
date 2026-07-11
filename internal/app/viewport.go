package app

// nodeWindowInfo describes a clipped list viewport shared by Home and Rules.
type nodeWindowInfo struct {
	start    int
	end      int
	hasAbove bool
	hasBelow bool
}

func nodeWindow(bodyHeight, totalNodes, cursor, previousStart int) nodeWindowInfo {
	if totalNodes <= 0 {
		return nodeWindowInfo{}
	}
	if bodyHeight < 2 {
		bodyHeight = 2
	}
	cursor = min(max(cursor, 0), totalNodes-1)

	best := nodeWindowInfo{start: 0, end: min(totalNodes, bodyHeight-1)}
	bestScore := -1
	for start := 0; start < totalNodes; start++ {
		hasAbove := start > 0
		available := bodyHeight - 1
		if hasAbove {
			available--
		}
		available = max(available, 1)
		end := min(totalNodes, start+available)
		hasBelow := end < totalNodes
		if hasBelow {
			end = min(totalNodes, start+max(available-1, 1))
		}
		if cursor < start || cursor >= end {
			continue
		}

		score := end - start
		if hasAbove {
			score++
		}
		if hasBelow {
			score++
		}
		if score > bodyHeight-1 {
			continue
		}
		if score > bestScore || (score == bestScore && distance(start, previousStart) < distance(best.start, previousStart)) {
			best = nodeWindowInfo{start: start, end: end, hasAbove: hasAbove, hasBelow: hasBelow}
			bestScore = score
		}
	}
	return best
}

func distance(left, right int) int {
	if left < right {
		return right - left
	}
	return left - right
}
