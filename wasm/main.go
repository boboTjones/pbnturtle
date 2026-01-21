package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"syscall/js"
)

// ProcessResult contains the result of image processing
type ProcessResult struct {
	Image   string      `json:"image"`
	Palette []ColorInfo `json:"palette"`
	Error   string      `json:"error,omitempty"`
}

// ColorInfo contains color information
type ColorInfo struct {
	Number int    `json:"number"`
	Hex    string `json:"hex"`
	C      int    `json:"c"`
	M      int    `json:"m"`
	Y      int    `json:"y"`
	K      int    `json:"k"`
}

func main() {
	fmt.Println("🎨 Paint by Numbers WASM initialized!")

	// Register the main processing function
	js.Global().Set("processImage", js.FuncOf(processImage))

	// Keep the program running
	<-make(chan bool)
}

// processImage is called from JavaScript with image data and parameters
func processImage(this js.Value, args []js.Value) interface{} {
	if len(args) < 7 {
		return createErrorResult("Invalid arguments: expected (imageData, points, colors, lineWidth, maxDimension, showColors, useVoronoi)")
	}

	// Get arguments
	imageData := args[0]
	numPoints := args[1].Int()
	numColors := args[2].Int()
	lineWidth := args[3].Int()
	maxDimension := args[4].Int()
	showColors := args[5].Bool()
	useVoronoi := args[6].Bool()

	// Validate parameters
	if numPoints < 50 || numPoints > 50000 {
		return createErrorResult("Points must be between 50 and 50000")
	}
	if numColors < 2 || numColors > 64 {
		return createErrorResult("Colors must be between 2 and 64")
	}
	if lineWidth < 0 || lineWidth > 5 {
		return createErrorResult("Line width must be between 0 and 5")
	}
	if maxDimension < 256 || maxDimension > 4096 {
		return createErrorResult("Max dimension must be between 256 and 4096")
	}

	// Convert JavaScript Uint8Array to Go byte slice
	length := imageData.Get("length").Int()
	imageBytes := make([]byte, length)
	js.CopyBytesToGo(imageBytes, imageData)

	fmt.Printf("Processing: %d bytes, points=%d, colors=%d, lineWidth=%d, maxDim=%d, showColors=%v, voronoi=%v\n",
		length, numPoints, numColors, lineWidth, maxDimension, showColors, useVoronoi)

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to decode image: %v", err))
	}

	fmt.Printf("Decoded %s image: %dx%d\n", format, img.Bounds().Dx(), img.Bounds().Dy())

	// Downsample if needed
	img = downsampleImage(img, maxDimension)

	// Process image
	result, palette := convertToPaintByNumbersWithMode(img, numPoints, numColors, lineWidth, showColors, useVoronoi)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, result); err != nil {
		return createErrorResult(fmt.Sprintf("Failed to encode result: %v", err))
	}

	// Build palette info
	paletteInfo := make([]ColorInfo, len(palette))
	for i, c := range palette {
		cyan, magenta, yellow, black := rgbToCMYK(c)
		paletteInfo[i] = ColorInfo{
			Number: i + 1,
			Hex:    colorToHex(c),
			C:      cyan,
			M:      magenta,
			Y:      yellow,
			K:      black,
		}
	}

	// Create response
	response := ProcessResult{
		Image:   base64.StdEncoding.EncodeToString(buf.Bytes()),
		Palette: paletteInfo,
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to marshal JSON: %v", err))
	}

	fmt.Println("✓ Processing complete!")

	return string(jsonBytes)
}

func createErrorResult(errMsg string) interface{} {
	result := ProcessResult{Error: errMsg}
	jsonBytes, _ := json.Marshal(result)
	return string(jsonBytes)
}
