package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"sort"
)

// convertToPaintByNumbers converts an image to paint-by-numbers style using Voronoi diagrams
func convertToPaintByNumbers(img image.Image, numPoints, numColors int) image.Image {
	result, _ := convertToPaintByNumbersWithPalette(img, numPoints, numColors)
	return result
}

// convertToPaintByNumbersWithPalette converts an image and returns both the result and color palette
func convertToPaintByNumbersWithPalette(img image.Image, numPoints, numColors int) (image.Image, []color.Color) {
	return convertToPaintByNumbersWithProgress(img, numPoints, numColors, nil)
}

// convertToPaintByNumbersWithProgress converts with progress reporting
func convertToPaintByNumbersWithProgress(img image.Image, numPoints, numColors int, progress ProgressCallback) (image.Image, []color.Color) {
	bounds := img.Bounds()

	if progress != nil {
		progress("Generating color palette", 0)
	}

	// Step 1: Quantize colors - reduce to a palette (do this first to avoid redundant work)
	palette := generatePalette(img, numColors)

	// Step 2: Generate Voronoi points with adaptive distribution
	points := generateAdaptiveVoronoiPoints(img, numPoints, progress)

	if progress != nil {
		progress("Quantizing points", 20)
	}

	// Step 3: Map points to palette colors
	quantizedPoints := quantizePoints(points, palette)

	// Step 4: Create Voronoi diagram with quantized colors
	voronoi, kdtree := createVoronoiDiagramWithProgress(bounds, quantizedPoints, progress)

	if progress != nil {
		progress("Drawing borders", 70)
	}

	// Step 5: Add borders between regions
	result := addVoronoiBorders(voronoi, quantizedPoints)

	if progress != nil {
		progress("Adding numbers", 85)
	}

	// Step 6: Add color numbers to regions
	result = addRegionNumbers(result, quantizedPoints, kdtree)

	if progress != nil {
		progress("Complete", 100)
	}

	return result, palette
}

// generatePalette generates a color palette from the image using k-means clustering
func generatePalette(img image.Image, numColors int) []color.Color {
	bounds := img.Bounds()

	// Sample colors from the image
	var colors []color.Color
	sampleStep := 10
	for y := bounds.Min.Y; y < bounds.Max.Y; y += sampleStep {
		for x := bounds.Min.X; x < bounds.Max.X; x += sampleStep {
			colors = append(colors, img.At(x, y))
		}
	}

	// Simple k-means clustering to find representative colors
	return kMeansClustering(colors, numColors)
}

// kMeansClustering performs k-means clustering on colors with k-means++ initialization
func kMeansClustering(colors []color.Color, k int) []color.Color {
	if len(colors) == 0 {
		return []color.Color{color.RGBA{128, 128, 128, 255}}
	}

	if k >= len(colors) {
		return colors
	}

	// K-means++ initialization for better centroids
	centroids := make([]color.Color, 0, k)

	// Choose first centroid randomly
	centroids = append(centroids, colors[rand.Intn(len(colors))])

	// Choose remaining centroids with probability proportional to distance squared
	for len(centroids) < k {
		distances := make([]float64, len(colors))
		totalDist := 0.0

		for i, c := range colors {
			minDist := math.MaxFloat64
			for _, centroid := range centroids {
				dist := colorDistanceSquared(c, centroid)
				if dist < minDist {
					minDist = dist
				}
			}
			distances[i] = minDist
			totalDist += minDist
		}

		// Select next centroid with weighted probability
		target := rand.Float64() * totalDist
		cumulative := 0.0
		for i, dist := range distances {
			cumulative += dist
			if cumulative >= target {
				centroids = append(centroids, colors[i])
				break
			}
		}
	}

	// Run k-means iterations
	for iter := 0; iter < 15; iter++ {
		// Assign each color to nearest centroid
		clusters := make([][]color.Color, k)
		for _, c := range colors {
			nearest := findNearestColor(c, centroids)
			clusters[nearest] = append(clusters[nearest], c)
		}

		// Update centroids
		changed := false
		for i, cluster := range clusters {
			if len(cluster) > 0 {
				newCentroid := averageColor(cluster)
				if !colorsEqual(centroids[i], newCentroid) {
					centroids[i] = newCentroid
					changed = true
				}
			}
		}

		// Early stopping if converged
		if !changed {
			break
		}
	}

	return centroids
}

// colorDistanceSquared calculates squared color distance
func colorDistanceSquared(c1, c2 color.Color) float64 {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()

	dr := float64(r1) - float64(r2)
	dg := float64(g1) - float64(g2)
	db := float64(b1) - float64(b2)

	return dr*dr + dg*dg + db*db
}

// colorsEqual checks if two colors are equal
func colorsEqual(c1, c2 color.Color) bool {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

// findNearestColor finds the nearest color in the palette
func findNearestColor(c color.Color, palette []color.Color) int {
	r1, g1, b1, _ := c.RGBA()
	minDist := math.MaxFloat64
	nearest := 0

	for i, p := range palette {
		r2, g2, b2, _ := p.RGBA()
		dr := float64(r1) - float64(r2)
		dg := float64(g1) - float64(g2)
		db := float64(b1) - float64(b2)
		dist := dr*dr + dg*dg + db*db

		if dist < minDist {
			minDist = dist
			nearest = i
		}
	}

	return nearest
}

// averageColor computes the average of a set of colors
func averageColor(colors []color.Color) color.Color {
	var r, g, b, a uint64
	for _, c := range colors {
		cr, cg, cb, ca := c.RGBA()
		r += uint64(cr)
		g += uint64(cg)
		b += uint64(cb)
		a += uint64(ca)
	}

	n := uint64(len(colors))
	return color.RGBA{
		R: uint8((r / n) >> 8),
		G: uint8((g / n) >> 8),
		B: uint8((b / n) >> 8),
		A: uint8((a / n) >> 8),
	}
}

// quantizePoints maps each point's color to the nearest palette color
func quantizePoints(points []Point, palette []color.Color) []Point {
	quantized := make([]Point, len(points))
	for i, p := range points {
		nearest := findNearestColor(p.Color, palette)
		quantized[i] = Point{
			X:     p.X,
			Y:     p.Y,
			Color: palette[nearest],
			Index: p.Index,
		}
	}
	return quantized
}

// addVoronoiBorders adds black borders between Voronoi regions
func addVoronoiBorders(img *image.RGBA, points []Point) *image.RGBA {
	bounds := img.Bounds()
	result := image.NewRGBA(bounds)
	draw.Draw(result, bounds, img, bounds.Min, draw.Src)

	// For each pixel, check if neighbors belong to different regions
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if isBorderPixel(x, y, img, points) {
				result.Set(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	return result
}

// isBorderPixel checks if a pixel is on the border between regions
func isBorderPixel(x, y int, img *image.RGBA, points []Point) bool {
	bounds := img.Bounds()
	current := findNearestPoint(x, y, points)

	// Only check right and down neighbors to create thinner lines
	// This creates a single-pixel border on one side of each boundary
	neighbors := [][2]int{
		{x + 1, y}, // right
		{x, y + 1}, // down
	}

	for _, n := range neighbors {
		nx, ny := n[0], n[1]
		if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
			continue
		}

		neighbor := findNearestPoint(nx, ny, points)
		if neighbor != current {
			return true
		}
	}

	return false
}

// ColorDistance calculates the Euclidean distance between two colors
func colorDistance(c1, c2 color.Color) float64 {
	r1, g1, b1, _ := c1.RGBA()
	r2, g2, b2, _ := c2.RGBA()

	dr := float64(r1) - float64(r2)
	dg := float64(g1) - float64(g2)
	db := float64(b1) - float64(b2)

	return math.Sqrt(dr*dr + dg*dg + db*db)
}

// SortColorsByBrightness sorts colors by their perceived brightness
func sortColorsByBrightness(colors []color.Color) []color.Color {
	sorted := make([]color.Color, len(colors))
	copy(sorted, colors)

	sort.Slice(sorted, func(i, j int) bool {
		r1, g1, b1, _ := sorted[i].RGBA()
		r2, g2, b2, _ := sorted[j].RGBA()

		// Perceived brightness formula
		brightness1 := 0.299*float64(r1) + 0.587*float64(g1) + 0.114*float64(b1)
		brightness2 := 0.299*float64(r2) + 0.587*float64(g2) + 0.114*float64(b2)

		return brightness1 < brightness2
	})

	return sorted
}
