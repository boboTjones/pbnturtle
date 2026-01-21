package main

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"sync"
)

// Point represents a 2D point with an associated color
type Point struct {
	X, Y  int
	Color color.Color
	Index int // Index in the points array
}

// ProgressCallback is called to report progress
type ProgressCallback func(stage string, percent int)

// generateVoronoiPoints generates random points across the image
// and samples the color from the original image at those points
func generateVoronoiPoints(img image.Image, numPoints int) []Point {
	return generateAdaptiveVoronoiPoints(img, numPoints, nil)
}

// generateAdaptiveVoronoiPoints uses edge detection to place more points in high-detail areas
func generateAdaptiveVoronoiPoints(img image.Image, numPoints int, progress ProgressCallback) []Point {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if progress != nil {
		progress("Detecting edges", 5)
	}

	// Compute edge map
	edgeMap := computeEdgeMap(img)

	// Build cumulative distribution for weighted sampling
	totalWeight := 0.0
	weights := make([]float64, width*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			// Higher edge strength = higher weight
			weight := 1.0 + edgeMap[idx]*10.0 // Bias toward edges
			weights[idx] = weight
			totalWeight += weight
		}
	}

	// Convert to cumulative distribution
	cumulative := make([]float64, len(weights))
	sum := 0.0
	for i, w := range weights {
		sum += w
		cumulative[i] = sum
	}

	if progress != nil {
		progress("Sampling points", 15)
	}

	// Sample points using weighted distribution
	points := make([]Point, numPoints)
	for i := 0; i < numPoints; i++ {
		// Binary search to find weighted random position
		target := rand.Float64() * totalWeight
		idx := binarySearch(cumulative, target)

		x := (idx % width) + bounds.Min.X
		y := (idx / width) + bounds.Min.Y

		points[i] = Point{
			X:     x,
			Y:     y,
			Color: img.At(x, y),
			Index: i,
		}
	}

	return points
}

// computeEdgeMap uses Sobel operator for edge detection
func computeEdgeMap(img image.Image) []float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	edgeMap := make([]float64, width*height)

	// Sobel kernels
	sobelX := [][]int{
		{-1, 0, 1},
		{-2, 0, 2},
		{-1, 0, 1},
	}
	sobelY := [][]int{
		{-1, -2, -1},
		{0, 0, 0},
		{1, 2, 1},
	}

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			var gx, gy float64

			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					r, g, b, _ := img.At(x+dx+bounds.Min.X, y+dy+bounds.Min.Y).RGBA()
					// Convert to grayscale
					gray := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)

					gx += gray * float64(sobelX[dy+1][dx+1])
					gy += gray * float64(sobelY[dy+1][dx+1])
				}
			}

			// Gradient magnitude
			magnitude := math.Sqrt(gx*gx + gy*gy)
			edgeMap[y*width+x] = magnitude / 65535.0 // Normalize
		}
	}

	return edgeMap
}

// binarySearch finds the index where value would be inserted
func binarySearch(arr []float64, value float64) int {
	left, right := 0, len(arr)-1

	for left < right {
		mid := (left + right) / 2
		if arr[mid] < value {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left
}

// findNearestPoint finds the nearest Voronoi point to the given coordinates
func findNearestPoint(x, y int, points []Point) int {
	minDist := math.MaxFloat64
	nearest := 0

	for i, p := range points {
		dx := float64(x - p.X)
		dy := float64(y - p.Y)
		dist := dx*dx + dy*dy

		if dist < minDist {
			minDist = dist
			nearest = i
		}
	}

	return nearest
}

// createVoronoiDiagram creates a Voronoi diagram from the given points
func createVoronoiDiagram(bounds image.Rectangle, points []Point) (*image.RGBA, *KDTree) {
	return createVoronoiDiagramWithProgress(bounds, points, nil)
}

// createVoronoiDiagramWithProgress creates a Voronoi diagram with progress reporting
func createVoronoiDiagramWithProgress(bounds image.Rectangle, points []Point, progress ProgressCallback) (*image.RGBA, *KDTree) {
	img := image.NewRGBA(bounds)

	if progress != nil {
		progress("Building spatial index", 25)
	}

	// Build k-d tree for fast nearest neighbor queries
	kdtree := NewKDTree(points)

	if progress != nil {
		progress("Creating regions", 30)
	}

	// Parallelize row processing
	height := bounds.Dy()
	numWorkers := 8
	rowsPerWorker := (height + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	progressChan := make(chan int, numWorkers)

	// Progress tracking goroutine
	if progress != nil {
		go func() {
			completed := 0
			for range progressChan {
				completed++
				percent := 30 + (completed*40)/numWorkers
				progress("Creating regions", percent)
			}
		}()
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		startY := w * rowsPerWorker
		endY := (w + 1) * rowsPerWorker
		if endY > height {
			endY = height
		}

		go func(sy, ey int) {
			defer wg.Done()

			for y := sy; y < ey; y++ {
				for x := 0; x < bounds.Dx(); x++ {
					actualX := x + bounds.Min.X
					actualY := y + bounds.Min.Y
					nearestIdx := kdtree.FindNearest(actualX, actualY)
					img.Set(actualX, actualY, points[nearestIdx].Color)
				}
			}

			if progress != nil {
				progressChan <- 1
			}
		}(startY, endY)
	}

	wg.Wait()
	if progress != nil {
		close(progressChan)
	}

	return img, kdtree
}
