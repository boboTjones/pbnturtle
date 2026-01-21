// Web Worker for WASM processing
importScripts('wasm_exec.js');

let wasmReady = false;
let go = new Go();

// Load WASM
WebAssembly.instantiateStreaming(fetch('paintbynumbers.wasm'), go.importObject)
    .then((result) => {
        go.run(result.instance);
        wasmReady = true;
        self.postMessage({ type: 'ready' });
        console.log('Worker: WASM loaded');
    })
    .catch((err) => {
        self.postMessage({ type: 'error', error: 'Failed to load WASM: ' + err.message });
    });

// Listen for messages from main thread
self.onmessage = function(e) {
    if (e.data.type === 'process') {
        if (!wasmReady) {
            self.postMessage({ type: 'error', error: 'WASM not ready' });
            return;
        }

        const { imageData, points, colors, lineWidth, maxDimension, showColors, mode } = e.data;

        try {
            // Call Go WASM function
            const useVoronoi = mode === 'voronoi';
            const resultJSON = processImage(imageData, points, colors, lineWidth, maxDimension, showColors, useVoronoi);
            const result = JSON.parse(resultJSON);

            if (result.error) {
                self.postMessage({ type: 'error', error: result.error });
            } else {
                self.postMessage({ type: 'complete', result: result });
            }
        } catch (err) {
            self.postMessage({ type: 'error', error: err.message });
        }
    }
};
