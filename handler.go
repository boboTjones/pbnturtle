package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	"image/png"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	maxPoints      = 50000
	maxColors      = 64
	maxDimension   = 4096
	requestTimeout = 5 * time.Minute
)

type ColorInfo struct {
	Number int    `json:"number"`
	Hex    string `json:"hex"`
	C      int    `json:"c"` // Cyan %
	M      int    `json:"m"` // Magenta %
	Y      int    `json:"y"` // Yellow %
	K      int    `json:"k"` // Black %
}

type ConvertResponse struct {
	Image   string      `json:"image"`   // base64 encoded PNG
	Palette []ColorInfo `json:"palette"` // color information
}

type ProgressEvent struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if client wants SSE progress updates
	acceptHeader := r.Header.Get("Accept")
	wantsSSE := acceptHeader == "text/event-stream"

	if wantsSSE {
		handleConvertSSE(w, r)
	} else {
		handleConvertJSON(w, r)
	}
}

func handleConvertJSON(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Parse and process
	img, numPoints, numColors, err := parseRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Process with timeout
	resultChan := make(chan struct {
		result  image.Image
		palette []color.Color
		err     error
	}, 1)

	go func() {
		result, palette := convertToPaintByNumbersWithProgress(img, numPoints, numColors, nil)
		resultChan <- struct {
			result  image.Image
			palette []color.Color
			err     error
		}{result, palette, nil}
	}()

	select {
	case <-ctx.Done():
		http.Error(w, "Request timeout", http.StatusRequestTimeout)
		return
	case res := <-resultChan:
		if res.err != nil {
			http.Error(w, res.err.Error(), http.StatusInternalServerError)
			return
		}

		sendJSONResponse(w, res.result, res.palette)
	}
}

func handleConvertSSE(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Parse request
	img, numPoints, numColors, err := parseRequest(r)
	if err != nil {
		sendSSEError(w, err.Error(), flusher)
		return
	}

	// Progress callback
	var progressMu sync.Mutex
	progressCallback := func(stage string, percent int) {
		progressMu.Lock()
		defer progressMu.Unlock()

		event := ProgressEvent{Stage: stage, Percent: percent}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Process with progress
	resultChan := make(chan struct {
		result  image.Image
		palette []color.Color
		err     error
	}, 1)

	go func() {
		result, palette := convertToPaintByNumbersWithProgress(img, numPoints, numColors, progressCallback)
		resultChan <- struct {
			result  image.Image
			palette []color.Color
			err     error
		}{result, palette, nil}
	}()

	select {
	case <-ctx.Done():
		sendSSEError(w, "Request timeout", flusher)
		return
	case res := <-resultChan:
		if res.err != nil {
			sendSSEError(w, res.err.Error(), flusher)
			return
		}

		// Encode result
		var buf bytes.Buffer
		if err := png.Encode(&buf, res.result); err != nil {
			sendSSEError(w, "Failed to encode result", flusher)
			return
		}

		paletteInfo := make([]ColorInfo, len(res.palette))
		for i, c := range res.palette {
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

		response := ConvertResponse{
			Image:   base64.StdEncoding.EncodeToString(buf.Bytes()),
			Palette: paletteInfo,
		}

		data, _ := json.Marshal(response)
		// SSE requires each line of data to be prefixed with "data: "
		// Split the JSON and prefix each line
		dataStr := string(data)
		fmt.Fprintf(w, "event: done\ndata: %s\n\n", dataStr)
		flusher.Flush()
	}
}

func parseRequest(r *http.Request) (image.Image, int, int, error) {
	// Parse multipart form
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		return nil, 0, 0, fmt.Errorf("failed to parse form: %w", err)
	}

	// Get the image file
	file, header, err := r.FormFile("image")
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get image: %w", err)
	}
	defer file.Close()

	log.Printf("Processing image: %s (%d bytes)", header.Filename, header.Size)

	// Decode the image
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	log.Printf("Image decoded: format=%s, size=%dx%d", format, img.Bounds().Dx(), img.Bounds().Dy())

	// Get max dimension parameter
	maxDim := 2048
	if maxDimStr := r.FormValue("maxDimension"); maxDimStr != "" {
		if md, err := strconv.Atoi(maxDimStr); err == nil && md > 0 && md <= maxDimension {
			maxDim = md
		}
	}

	// Downsample if needed
	originalSize := img.Bounds().Dx() * img.Bounds().Dy()
	img = downsampleImage(img, maxDim)
	newSize := img.Bounds().Dx() * img.Bounds().Dy()

	if newSize < originalSize {
		log.Printf("Image downsampled: %dx%d -> %dx%d",
			img.Bounds().Dx(), img.Bounds().Dy(),
			img.Bounds().Dx(), img.Bounds().Dy())
	}

	// Get parameters with limits
	numPoints := 2000
	numColors := 12

	if pointsStr := r.FormValue("points"); pointsStr != "" {
		if p, err := strconv.Atoi(pointsStr); err == nil && p > 0 {
			if p > maxPoints {
				numPoints = maxPoints
			} else {
				numPoints = p
			}
		}
	}

	if colorsStr := r.FormValue("colors"); colorsStr != "" {
		if c, err := strconv.Atoi(colorsStr); err == nil && c > 0 {
			if c > maxColors {
				numColors = maxColors
			} else {
				numColors = c
			}
		}
	}

	log.Printf("Parameters: points=%d, colors=%d, maxDim=%d", numPoints, numColors, maxDim)

	return img, numPoints, numColors, nil
}

func sendJSONResponse(w http.ResponseWriter, result image.Image, palette []color.Color) {
	// Encode image to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, result); err != nil {
		http.Error(w, "Failed to encode result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert palette to ColorInfo
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

	// Create response with base64 image and palette
	response := ConvertResponse{
		Image:   base64.StdEncoding.EncodeToString(buf.Bytes()),
		Palette: paletteInfo,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to write response: %v", err)
		return
	}

	log.Println("Image processed successfully")
}

func sendSSEError(w http.ResponseWriter, message string, flusher http.Flusher) {
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", message)
	flusher.Flush()
}

func colorToHex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
}

// rgbToCMYK converts RGB color to CMYK percentages
func rgbToCMYK(col color.Color) (int, int, int, int) {
	r, g, b, _ := col.RGBA()

	// Convert to 0-1 range
	rFloat := float64(r) / 65535.0
	gFloat := float64(g) / 65535.0
	bFloat := float64(b) / 65535.0

	// Calculate K (black)
	k := 1.0 - max(rFloat, max(gFloat, bFloat))

	// If k is 1 (pure black), C, M, Y are 0
	if k >= 1.0 {
		return 0, 0, 0, 100
	}

	// Calculate C, M, Y
	cyan := (1.0 - rFloat - k) / (1.0 - k)
	magenta := (1.0 - gFloat - k) / (1.0 - k)
	yellow := (1.0 - bFloat - k) / (1.0 - k)

	// Convert to percentages
	return int(cyan * 100), int(magenta * 100), int(yellow * 100), int(k * 100)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
