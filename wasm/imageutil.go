package main

import (
	"image"
	"image/color"
)

// downsampleImage resizes an image to fit within maxDimension while preserving aspect ratio
func downsampleImage(img image.Image, maxDimension int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Check if downsampling is needed
	if width <= maxDimension && height <= maxDimension {
		return img
	}

	// Calculate new dimensions
	var newWidth, newHeight int
	if width > height {
		newWidth = maxDimension
		newHeight = (height * maxDimension) / width
	} else {
		newHeight = maxDimension
		newWidth = (width * maxDimension) / height
	}

	// Use bilinear interpolation for downsampling
	return resizeBilinear(img, newWidth, newHeight)
}

// resizeBilinear performs bilinear interpolation resizing
func resizeBilinear(img image.Image, newWidth, newHeight int) image.Image {
	bounds := img.Bounds()
	oldWidth := bounds.Dx()
	oldHeight := bounds.Dy()

	result := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	xRatio := float64(oldWidth) / float64(newWidth)
	yRatio := float64(oldHeight) / float64(newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Calculate source position
			srcX := float64(x) * xRatio
			srcY := float64(y) * yRatio

			// Get the four surrounding pixels
			x1 := int(srcX)
			y1 := int(srcY)
			x2 := x1 + 1
			y2 := y1 + 1

			// Clamp to image bounds
			if x2 >= oldWidth {
				x2 = oldWidth - 1
			}
			if y2 >= oldHeight {
				y2 = oldHeight - 1
			}

			// Get colors of surrounding pixels
			c11 := img.At(x1+bounds.Min.X, y1+bounds.Min.Y)
			c12 := img.At(x1+bounds.Min.X, y2+bounds.Min.Y)
			c21 := img.At(x2+bounds.Min.X, y1+bounds.Min.Y)
			c22 := img.At(x2+bounds.Min.X, y2+bounds.Min.Y)

			// Calculate interpolation weights
			xWeight := srcX - float64(x1)
			yWeight := srcY - float64(y1)

			// Interpolate
			interpolated := bilinearInterpolate(c11, c12, c21, c22, xWeight, yWeight)
			result.Set(x, y, interpolated)
		}
	}

	return result
}

// bilinearInterpolate interpolates between four colors
func bilinearInterpolate(c11, c12, c21, c22 color.Color, xWeight, yWeight float64) color.Color {
	r11, g11, b11, a11 := c11.RGBA()
	r12, g12, b12, a12 := c12.RGBA()
	r21, g21, b21, a21 := c21.RGBA()
	r22, g22, b22, a22 := c22.RGBA()

	// Interpolate along X axis
	r1 := interpolate(float64(r11), float64(r21), xWeight)
	g1 := interpolate(float64(g11), float64(g21), xWeight)
	b1 := interpolate(float64(b11), float64(b21), xWeight)
	a1 := interpolate(float64(a11), float64(a21), xWeight)

	r2 := interpolate(float64(r12), float64(r22), xWeight)
	g2 := interpolate(float64(g12), float64(g22), xWeight)
	b2 := interpolate(float64(b12), float64(b22), xWeight)
	a2 := interpolate(float64(a12), float64(a22), xWeight)

	// Interpolate along Y axis
	r := interpolate(r1, r2, yWeight)
	g := interpolate(g1, g2, yWeight)
	b := interpolate(b1, b2, yWeight)
	a := interpolate(a1, a2, yWeight)

	return color.RGBA{
		R: uint8(uint64(r) >> 8),
		G: uint8(uint64(g) >> 8),
		B: uint8(uint64(b) >> 8),
		A: uint8(uint64(a) >> 8),
	}
}

// interpolate performs linear interpolation
func interpolate(v1, v2, weight float64) float64 {
	return v1*(1-weight) + v2*weight
}
