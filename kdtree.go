package main

// KDTree implements a 2D k-d tree for fast nearest neighbor search
type KDTree struct {
	root *kdNode
}

type kdNode struct {
	point     Point
	left      *kdNode
	right     *kdNode
	splitAxis int // 0 for X, 1 for Y
}

// NewKDTree builds a k-d tree from a slice of points
func NewKDTree(points []Point) *KDTree {
	if len(points) == 0 {
		return &KDTree{}
	}

	// Make a copy to avoid modifying the original
	pointsCopy := make([]Point, len(points))
	copy(pointsCopy, points)

	return &KDTree{
		root: buildKDTree(pointsCopy, 0),
	}
}

// buildKDTree recursively builds the k-d tree
func buildKDTree(points []Point, depth int) *kdNode {
	if len(points) == 0 {
		return nil
	}

	if len(points) == 1 {
		return &kdNode{
			point:     points[0],
			splitAxis: depth % 2,
		}
	}

	axis := depth % 2

	// Sort points by the current axis
	sortPointsByAxis(points, axis)

	// Find median
	median := len(points) / 2

	return &kdNode{
		point:     points[median],
		splitAxis: axis,
		left:      buildKDTree(points[:median], depth+1),
		right:     buildKDTree(points[median+1:], depth+1),
	}
}

// sortPointsByAxis sorts points by X (axis=0) or Y (axis=1)
func sortPointsByAxis(points []Point, axis int) {
	// Simple insertion sort (efficient for small arrays)
	for i := 1; i < len(points); i++ {
		key := points[i]
		j := i - 1

		var keyVal, pointVal int
		if axis == 0 {
			keyVal = key.X
		} else {
			keyVal = key.Y
		}

		for j >= 0 {
			if axis == 0 {
				pointVal = points[j].X
			} else {
				pointVal = points[j].Y
			}

			if pointVal <= keyVal {
				break
			}

			points[j+1] = points[j]
			j--
		}
		points[j+1] = key
	}
}

// FindNearest returns the index of the nearest point to (x, y)
func (tree *KDTree) FindNearest(x, y int) int {
	if tree.root == nil {
		return 0
	}

	bestNode := tree.root
	bestDist := distanceSquared(x, y, tree.root.point.X, tree.root.point.Y)

	tree.findNearestHelper(tree.root, x, y, &bestNode, &bestDist)

	return bestNode.point.Index
}

// findNearestHelper recursively searches for the nearest point
func (tree *KDTree) findNearestHelper(node *kdNode, x, y int, bestNode **kdNode, bestDist *float64) {
	if node == nil {
		return
	}

	// Check if current node is closer
	dist := distanceSquared(x, y, node.point.X, node.point.Y)
	if dist < *bestDist {
		*bestDist = dist
		*bestNode = node
	}

	// Determine which side to search first
	var nearChild, farChild *kdNode
	var diff int

	if node.splitAxis == 0 {
		diff = x - node.point.X
	} else {
		diff = y - node.point.Y
	}

	if diff < 0 {
		nearChild = node.left
		farChild = node.right
	} else {
		nearChild = node.right
		farChild = node.left
	}

	// Search near side
	tree.findNearestHelper(nearChild, x, y, bestNode, bestDist)

	// Check if we need to search far side
	// Only search if the splitting plane is closer than current best
	if float64(diff*diff) < *bestDist {
		tree.findNearestHelper(farChild, x, y, bestNode, bestDist)
	}
}

// distanceSquared calculates squared Euclidean distance
func distanceSquared(x1, y1, x2, y2 int) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return dx*dx + dy*dy
}
