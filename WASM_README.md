# Paint by Numbers - WebAssembly Edition

A fully client-side paint-by-numbers generator that runs entirely in your browser using WebAssembly!

## Features

- **100% Client-Side** - No server required, all processing happens in your browser
- **Real-Time Sliders** - Adjust points, colors, and line width with instant preview
- **WebAssembly Performance** - Near-native speed using compiled Go code
- **Drag & Drop** - Easy image upload
- **Auto-Update Mode** - Toggle automatic processing on slider change
- **Download Results** - Save your paint-by-numbers as PNG

## Quick Start

### Option 1: Use the existing server

The HTTP server is already running on port 8081:

```bash
# Just open your browser to:
http://localhost:8081/
```

### Option 2: Start a new server

```bash
cd /Users/erin/codebase/trdlz
python3 -m http.server 8081
```

Then open: <http://localhost:8081/>

## How to Use

1. **Upload Image**
   - Click the drop zone or drag & drop an image
   - Supports JPEG, PNG, and GIF

2. **Adjust Parameters**
   - **Voronoi Points** (50-10,000): More points = more detail
   - **Colors** (2-64): Fewer colors = more stylized
   - **Line Width** (0-5): Border thickness (0 = no borders)

3. **Process**
   - **Auto-update ON**: Changes apply automatically after 500ms
   - **Auto-update OFF**: Click "Process Image" button manually

4. **Download**
   - Click the download button to save your result

## Controls Explained

### Voronoi Points

- **50-500**: Very stylized, abstract look
- **500-2000**: Classic paint-by-numbers style (default: 2000)
- **2000-10000**: High detail, closer to original

### Colors

- **2-6**: Minimal palette, bold look
- **8-16**: Balanced (default: 12)
- **16-64**: Many shades, subtle transitions

### Line Width

- **0**: No borders (pure color regions)
- **1**: Thin lines (default)
- **2-3**: Medium borders
- **4-5**: Thick borders (numbers may not fit)

## Building from Source

```bash
cd wasm
GOOS=js GOARCH=wasm go build -o ../paintbynumbers.wasm .
```

## File Structure

```plaintext
trdlz/
├── index.html              # Main UI
├── wasm_exec.js           # Go WASM runtime
├── paintbynumbers.wasm    # Compiled WASM binary (3.6MB)
└── wasm/                  # Source code
    ├── main.go            # WASM interface
    ├── wasm_helpers.go    # Line width support
    ├── voronoi.go         # Voronoi generation
    ├── paintbynumbers.go  # Color quantization
    ├── kdtree.go          # Spatial indexing
    ├── imageutil.go       # Image processing
    └── textdraw.go        # Region numbering
```

## Performance

- **Initial Load**: ~3.6MB WASM download (one-time)
- **Processing Time**:
  - Small images (500x500): ~1-2 seconds
  - Medium images (1024x1024): ~3-5 seconds
  - Large images (2048x2048): ~8-12 seconds
- **Optimizations**:
  - K-D tree spatial indexing (765x faster)
  - Parallel processing (8 workers)
  - Edge-adaptive point distribution
  - K-means++ color clustering

## Browser Support

Requires a modern browser with WebAssembly support:

- ✅ Chrome 57+
- ✅ Firefox 52+
- ✅ Safari 11+
- ✅ Edge 16+

## Advantages Over Server Version

1. **No Server Traffic** - Everything runs locally
2. **Instant Feedback** - Real-time slider updates
3. **Privacy** - Images never leave your computer
4. **Offline Support** - Works without internet (after initial load)
5. **Cost-Free** - No server hosting costs

## Tips

- Use **Auto-update OFF** for very large images to avoid constant re-processing
- Start with low points/colors, then increase for final render
- Line width 0 creates pure Voronoi art (no paint-by-numbers lines)
- The app automatically downsamples images larger than 2048px

Enjoy creating paint-by-numbers art!
