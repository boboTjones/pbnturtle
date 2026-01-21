package main

import (
	"image"
	"image/color"
	"image/draw"
)

// Simple 5x7 bitmap font for digits
var digitBitmaps = map[rune][][]bool{
	'0': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, true, true},
		{true, false, true, false, true},
		{true, true, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'1': {
		{false, false, true, false, false},
		{false, true, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, true, true, true, false},
	},
	'2': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{false, false, false, false, true},
		{false, false, false, true, false},
		{false, false, true, false, false},
		{false, true, false, false, false},
		{true, true, true, true, true},
	},
	'3': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{false, false, false, false, true},
		{false, false, true, true, false},
		{false, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'4': {
		{false, false, false, true, false},
		{false, false, true, true, false},
		{false, true, false, true, false},
		{true, false, false, true, false},
		{true, true, true, true, true},
		{false, false, false, true, false},
		{false, false, false, true, false},
	},
	'5': {
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, true, true, true, false},
		{false, false, false, false, true},
		{false, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'6': {
		{false, false, true, true, false},
		{false, true, false, false, false},
		{true, false, false, false, false},
		{true, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'7': {
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, true, false},
		{false, false, true, false, false},
		{false, true, false, false, false},
		{false, true, false, false, false},
		{false, true, false, false, false},
	},
	'8': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, false},
	},
	'9': {
		{false, true, true, true, false},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{false, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, true, false},
		{false, true, true, false, false},
	},
}

// Region represents a connected area in the image
type Region struct {
	ColorIndex int
	Pixels     []image.Point
	Centroid   image.Point
	Area       int
}

// findRegions identifies connected regions for each color
func findRegions(img *image.RGBA, points []Point, kdtree *KDTree) []Region {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Build region map
	visited := make([]bool, width*height)
	var regions []Region

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			idx := (y-bounds.Min.Y)*width + (x - bounds.Min.X)
			if visited[idx] {
				continue
			}

			// Start new region
			colorIdx := kdtree.FindNearest(x, y)
			region := floodFill(img, x, y, colorIdx, visited, kdtree, bounds)

			// Only keep regions with reasonable size
			if len(region.Pixels) >= 100 {
				// Calculate centroid
				sumX, sumY := 0, 0
				for _, p := range region.Pixels {
					sumX += p.X
					sumY += p.Y
				}
				region.Centroid = image.Point{
					X: sumX / len(region.Pixels),
					Y: sumY / len(region.Pixels),
				}
				region.Area = len(region.Pixels)
				regions = append(regions, region)
			}
		}
	}

	return regions
}

// floodFill performs flood fill to identify a connected region
func floodFill(img *image.RGBA, startX, startY, colorIdx int, visited []bool, kdtree *KDTree, bounds image.Rectangle) Region {
	width := bounds.Dx()
	queue := []image.Point{{X: startX, Y: startY}}
	region := Region{ColorIndex: colorIdx}

	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]

		if p.X < bounds.Min.X || p.X >= bounds.Max.X || p.Y < bounds.Min.Y || p.Y >= bounds.Max.Y {
			continue
		}

		idx := (p.Y-bounds.Min.Y)*width + (p.X - bounds.Min.X)
		if visited[idx] {
			continue
		}

		currentColorIdx := kdtree.FindNearest(p.X, p.Y)
		if currentColorIdx != colorIdx {
			continue
		}

		visited[idx] = true
		region.Pixels = append(region.Pixels, p)

		// Add neighbors to queue
		queue = append(queue,
			image.Point{X: p.X - 1, Y: p.Y},
			image.Point{X: p.X + 1, Y: p.Y},
			image.Point{X: p.X, Y: p.Y - 1},
			image.Point{X: p.X, Y: p.Y + 1},
		)
	}

	return region
}

// drawNumber draws a number at the specified position with white text and black outline
func drawNumber(img *image.RGBA, num int, x, y int) {
	if num >= 10 {
		// For two-digit numbers, draw them side by side
		tens := num / 10
		ones := num % 10
		drawDigit(img, tens, x-3, y)
		drawDigit(img, ones, x+3, y)
	} else {
		drawDigit(img, num, x, y)
	}
}

// drawDigit draws a single digit with outline
func drawDigit(img *image.RGBA, digit int, centerX, centerY int) {
	if digit < 0 || digit > 9 {
		return
	}

	bitmap := digitBitmaps[rune('0'+digit)]
	if bitmap == nil {
		return
	}

	// Calculate top-left position (center the digit)
	startX := centerX - 2
	startY := centerY - 3

	// Draw black outline first (1 pixel border)
	outlineColor := color.RGBA{0, 0, 0, 255}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			drawBitmap(img, bitmap, startX+dx, startY+dy, outlineColor)
		}
	}

	// Draw white digit on top
	textColor := color.RGBA{255, 255, 255, 255}
	drawBitmap(img, bitmap, startX, startY, textColor)
}

// drawBitmap draws a bitmap at the specified position
func drawBitmap(img *image.RGBA, bitmap [][]bool, startX, startY int, c color.Color) {
	bounds := img.Bounds()
	for y, row := range bitmap {
		for x, pixel := range row {
			if pixel {
				px := startX + x
				py := startY + y
				if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
					img.Set(px, py, c)
				}
			}
		}
	}
}

// addRegionNumbers adds color numbers to each region
func addRegionNumbers(img *image.RGBA, points []Point, kdtree *KDTree) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, img.Bounds(), img, img.Bounds().Min, draw.Src)

	// Find all regions
	regions := findRegions(img, points, kdtree)

	// Draw numbers on each region
	for _, region := range regions {
		// Color numbers start at 1
		colorNumber := region.ColorIndex + 1
		drawNumber(result, colorNumber, region.Centroid.X, region.Centroid.Y)
	}

	return result
}
