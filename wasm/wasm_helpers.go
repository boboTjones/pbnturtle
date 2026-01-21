package main

import (
	"fmt"
	"image"
	"image/color"
)

// convertToPaintByNumbersWithParams is the main entry point with line width support
func convertToPaintByNumbersWithParams(img image.Image, numPoints, numColors, lineWidth int) (image.Image, []color.Color) {
	return convertToPaintByNumbersWithParamsAndColors(img, numPoints, numColors, lineWidth, true)
}

// convertToPaintByNumbersWithMode supports both Voronoi and Grid modes
func convertToPaintByNumbersWithMode(img image.Image, numPoints, numColors, lineWidth int, showColors bool, useVoronoi bool) (image.Image, []color.Color) {
	if useVoronoi {
		return convertToPaintByNumbersWithParamsAndColors(img, numPoints, numColors, lineWidth, showColors)
	}
	return convertToGridPaintByNumbers(img, numColors, lineWidth, showColors)
}

// convertToPaintByNumbersWithParamsAndColors allows toggling color display
func convertToPaintByNumbersWithParamsAndColors(img image.Image, numPoints, numColors, lineWidth int, showColors bool) (image.Image, []color.Color) {
	bounds := img.Bounds()

	// Step 1: Generate color palette
	palette := generatePalette(img, numColors)

	// Step 2: Generate Voronoi points with adaptive distribution
	points := generateAdaptiveVoronoiPoints(img, numPoints, nil)

	// Step 3: Quantize points to palette colors
	quantizedPoints := quantizePoints(points, palette)

	// Step 4: Create Voronoi diagram
	var voronoi *image.RGBA
	var kdtree *KDTree

	if showColors {
		// Normal colored version
		voronoi, kdtree = createVoronoiDiagramWithProgress(bounds, quantizedPoints, nil)
	} else {
		// White/blank version (for coloring in)
		voronoi, kdtree = createBlankVoronoiDiagram(bounds, quantizedPoints)
	}

	// Step 5: Add borders with specified width
	result := addVoronoiBordersWithWidth(voronoi, quantizedPoints, lineWidth)

	// Step 6: Add region numbers if there's space
	if lineWidth <= 2 {
		result = addRegionNumbers(result, quantizedPoints, kdtree)
	}

	return result, palette
}

// createBlankVoronoiDiagram creates a white diagram with regions defined but not colored
func createBlankVoronoiDiagram(bounds image.Rectangle, points []Point) (*image.RGBA, *KDTree) {
	img := image.NewRGBA(bounds)

	// Fill with white
	white := color.RGBA{255, 255, 255, 255}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Set(x, y, white)
		}
	}

	// Build k-d tree for region identification
	kdtree := NewKDTree(points)

	return img, kdtree
}

// addVoronoiBordersWithWidth adds borders with configurable width
func addVoronoiBordersWithWidth(img *image.RGBA, points []Point, width int) *image.RGBA {
	if width == 0 {
		return img // No borders
	}

	bounds := img.Bounds()
	result := image.NewRGBA(bounds)

	// Copy original image
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			result.Set(x, y, img.At(x, y))
		}
	}

	// Draw borders
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if isBorderPixelWithWidth(x, y, img, points, width) {
				result.Set(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}

	return result
}

// isBorderPixelWithWidth checks if pixel should be part of border with given width
func isBorderPixelWithWidth(x, y int, img *image.RGBA, points []Point, width int) bool {
	bounds := img.Bounds()
	current := findNearestPoint(x, y, points)

	// Check neighbors in a radius based on width
	radius := (width + 1) / 2

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			// Skip center pixel
			if dx == 0 && dy == 0 {
				continue
			}

			// Only check neighbors within radius
			if dx*dx+dy*dy > radius*radius {
				continue
			}

			nx, ny := x+dx, y+dy

			// Skip out of bounds
			if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
				continue
			}

			neighbor := findNearestPoint(nx, ny, points)
			if neighbor != current {
				return true
			}
		}
	}

	return false
}

// rgbToCMYK converts RGB to CMYK percentages
func rgbToCMYK(col color.Color) (int, int, int, int) {
	r, g, b, _ := col.RGBA()

	rFloat := float64(r) / 65535.0
	gFloat := float64(g) / 65535.0
	bFloat := float64(b) / 65535.0

	k := 1.0 - max(rFloat, max(gFloat, bFloat))

	if k >= 1.0 {
		return 0, 0, 0, 100
	}

	cyan := (1.0 - rFloat - k) / (1.0 - k)
	magenta := (1.0 - gFloat - k) / (1.0 - k)
	yellow := (1.0 - bFloat - k) / (1.0 - k)

	return int(cyan * 100), int(magenta * 100), int(yellow * 100), int(k * 100)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// convertToGridPaintByNumbers creates a grid-based paint by numbers (no voronoi)
func convertToGridPaintByNumbers(img image.Image, numColors, lineWidth int, showColors bool) (image.Image, []color.Color) {
	bounds := img.Bounds()

	// Step 1: Generate color palette
	palette := generatePalette(img, numColors)

	// Step 2: Quantize each pixel to nearest palette color
	quantized := image.NewRGBA(bounds)
	colorIndices := make([]int, bounds.Dx()*bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nearestIdx := findNearestColor(img.At(x, y), palette)
			colorIndices[(y-bounds.Min.Y)*bounds.Dx()+(x-bounds.Min.X)] = nearestIdx
			if showColors {
				quantized.Set(x, y, palette[nearestIdx])
			} else {
				quantized.Set(x, y, color.RGBA{255, 255, 255, 255}) // White
			}
		}
	}

	// Step 3: Add borders between different colors
	result := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			result.Set(x, y, quantized.At(x, y))
		}
	}

	if lineWidth > 0 {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				if isGridBorder(x, y, bounds, colorIndices, lineWidth) {
					result.Set(x, y, color.RGBA{0, 0, 0, 255})
				}
			}
		}
	}

	// Step 4: Add region numbers for small line widths
	if lineWidth <= 2 {
		result = addGridRegionNumbers(result, colorIndices, bounds)
	}

	return result, palette
}

// isGridBorder checks if a pixel should be a border in grid mode
func isGridBorder(x, y int, bounds image.Rectangle, colorIndices []int, width int) bool {
	w := bounds.Dx()
	currentIdx := (y-bounds.Min.Y)*w + (x - bounds.Min.X)
	if currentIdx < 0 || currentIdx >= len(colorIndices) {
		return false
	}
	current := colorIndices[currentIdx]

	radius := (width + 1) / 2
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if dx*dx+dy*dy > radius*radius {
				continue
			}

			nx, ny := x+dx, y+dy
			if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
				continue
			}

			neighborIdx := (ny-bounds.Min.Y)*w + (nx - bounds.Min.X)
			if neighborIdx >= 0 && neighborIdx < len(colorIndices) && colorIndices[neighborIdx] != current {
				return true
			}
		}
	}
	return false
}

// addGridRegionNumbers adds numbers to regions in grid mode
func addGridRegionNumbers(img *image.RGBA, colorIndices []int, bounds image.Rectangle) *image.RGBA {
	result := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			result.Set(x, y, img.At(x, y))
		}
	}

	// Find regions using flood fill
	visited := make([]bool, len(colorIndices))
	width := bounds.Dx()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			idx := (y-bounds.Min.Y)*width + (x - bounds.Min.X)
			if visited[idx] || idx >= len(colorIndices) {
				continue
			}

			// Find region
			pixels := gridFloodFill(x, y, colorIndices, visited, bounds)
			if len(pixels) >= 100 {
				// Calculate centroid
				sumX, sumY := 0, 0
				for _, p := range pixels {
					sumX += p.X
					sumY += p.Y
				}
				centerX := sumX / len(pixels)
				centerY := sumY / len(pixels)

				// Draw number (color index + 1)
				colorNumber := colorIndices[idx] + 1
				drawNumber(result, colorNumber, centerX, centerY)
			}
		}
	}

	return result
}

// gridFloodFill performs flood fill for grid regions
func gridFloodFill(startX, startY int, colorIndices []int, visited []bool, bounds image.Rectangle) []image.Point {
	width := bounds.Dx()
	startIdx := (startY-bounds.Min.Y)*width + (startX - bounds.Min.X)
	if startIdx >= len(colorIndices) {
		return nil
	}
	targetColor := colorIndices[startIdx]

	queue := []image.Point{{X: startX, Y: startY}}
	var pixels []image.Point

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		if p.X < bounds.Min.X || p.X >= bounds.Max.X || p.Y < bounds.Min.Y || p.Y >= bounds.Max.Y {
			continue
		}

		idx := (p.Y-bounds.Min.Y)*width + (p.X - bounds.Min.X)
		if idx >= len(colorIndices) || visited[idx] || colorIndices[idx] != targetColor {
			continue
		}

		visited[idx] = true
		pixels = append(pixels, p)

		queue = append(queue,
			image.Point{X: p.X - 1, Y: p.Y},
			image.Point{X: p.X + 1, Y: p.Y},
			image.Point{X: p.X, Y: p.Y - 1},
			image.Point{X: p.X, Y: p.Y + 1},
		)
	}

	return pixels
}
