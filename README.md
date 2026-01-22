# Paint by Numbers Turtle

A Go web service that converts images into paint-by-numbers style artwork using Voronoi diagrams. Currently deployed online at [https://pbnturtle.fly.dev/](https://pbnturtle.fly.dev/).

## Features

- Upload any image (JPEG, PNG, GIF)
- Converts to paint-by-numbers style using Voronoi tessellation
- Adjustable number of Voronoi points (controls detail level)
- Adjustable color palette size (controls number of colors)
- Black borders between regions for that classic paint-by-numbers look
- Displays extracted color palette with hex codes
- Single-page web interface with live results
- No image storage - processed in memory

## How It Works

1. **Voronoi Diagram Generation**: Random points are sampled from the uploaded image
2. **Color Quantization**: Image colors are reduced to a palette using k-means clustering
3. **Region Creation**: Each pixel is assigned to its nearest Voronoi point
4. **Border Drawing**: Black lines are drawn between adjacent regions

## Getting Started

### Run the server

```bash
go run .
```

The server will start on `http://localhost:8080`

### Use the web interface

1. Open your browser to `http://localhost:8080`
2. Adjust the parameters:
   - **Number of Points**: More points = more detail (50-5000, default: 2000)
   - **Number of Colors**: Fewer colors = more stylized (2-32, default: 12)
3. Select an image file
4. Click "Convert to Paint by Numbers"
5. View the result and color palette
6. Download your result

### API Usage

The API returns JSON with a base64-encoded image and color palette:

```bash
curl -X POST -F "image=@photo.jpg" -F "points=2000" -F "colors=12" \
  http://localhost:8080/convert
```

Response format:

```json
{
  "image": "iVBORw0KGgoAAAANSUhEUgAA...",
  "palette": ["#3a5f8c", "#e8d4b2", "#7a9e5d", ...]
}
```

## Configuration

- Default port: `8080` (modify in `main.go`)
- Max upload size: `10MB` (modify in `handler.go`)

## Project Structure

- `main.go` - HTTP server and web interface
- `handler.go` - Image upload and processing handler
- `voronoi.go` - Voronoi diagram generation
- `paintbynumbers.go` - Color quantization and paint-by-numbers conversion

## Examples

Try different parameters for different effects:

- **Low detail, few colors** (points=100, colors=6): Highly stylized, abstract look
- **Medium detail** (points=500, colors=12): Classic paint-by-numbers style
- **High detail** (points=2000, colors=24): More detailed, closer to original

## Technical Details

- Uses k-means clustering for color quantization
- Euclidean distance for nearest-neighbor calculations
- Border detection using 4-connected neighbor checking
- All processing done in memory - no file storage required
