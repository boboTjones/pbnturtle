package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/convert", handleConvert)

	port := "8080"
	fmt.Printf("Server starting on port %s...\n", port)
	fmt.Println("Upload images to http://localhost:8080/convert")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Paint by Numbers Converter</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
        }
        .upload-form {
            border: 2px dashed #ccc;
            padding: 40px;
            text-align: center;
            border-radius: 8px;
        }
        input[type="file"] {
            margin: 20px 0;
        }
        button {
            background-color: #4CAF50;
            color: white;
            padding: 12px 24px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        button:hover {
            background-color: #45a049;
        }
        #result {
            margin-top: 30px;
            text-align: center;
        }
        #result img {
            max-width: 100%;
            border: 1px solid #ddd;
            border-radius: 4px;
            padding: 5px;
        }
        .controls {
            margin: 20px 0;
        }
        label {
            margin-right: 10px;
        }
        input[type="number"] {
            width: 80px;
            padding: 5px;
        }
        .palette {
            margin: 30px 0;
            padding: 20px;
            background: #f9f9f9;
            border-radius: 8px;
        }
        .palette h3 {
            margin-top: 0;
        }
        .color-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
            gap: 15px;
            margin-top: 15px;
        }
        .color-item {
            text-align: center;
        }
        .color-number {
            font-weight: bold;
            font-size: 16px;
            color: #333;
            margin-bottom: 5px;
        }
        .color-swatch {
            width: 100%;
            height: 60px;
            border: 2px solid #333;
            border-radius: 4px;
            margin-bottom: 5px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .color-code {
            font-family: monospace;
            font-size: 11px;
            color: #666;
            margin-bottom: 5px;
        }
        .color-cmyk {
            font-family: monospace;
            font-size: 10px;
            color: #888;
            line-height: 1.3;
        }
        .download-btn {
            display: inline-block;
            margin: 20px 10px;
            padding: 10px 20px;
            background-color: #2196F3;
            color: white;
            text-decoration: none;
            border-radius: 4px;
        }
        .download-btn:hover {
            background-color: #0b7dda;
        }
        .progress-container {
            margin: 20px 0;
            display: none;
        }
        .progress-bar {
            width: 100%;
            height: 30px;
            background-color: #f0f0f0;
            border-radius: 15px;
            overflow: hidden;
            position: relative;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #4CAF50, #45a049);
            transition: width 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: bold;
        }
        .progress-text {
            text-align: center;
            margin-top: 10px;
            color: #666;
        }
    </style>
</head>
<body>
    <h1>Paint by Numbers Turtle</h1>
    <p>Upload an image to convert it into a paint-by-numbers style using Voronoi diagrams.</p>

    <div class="upload-form">
        <form id="uploadForm" enctype="multipart/form-data">
            <div class="controls">
                <label for="points">Number of Points:</label>
                <input type="number" id="points" name="points" value="2000" min="50" step="50">
            </div>
            <div class="controls">
                <label for="colors">Number of Colors:</label>
                <input type="number" id="colors" name="colors" value="12" min="2" max="64">
            </div>
            <div class="controls">
                <label for="maxDim">Max Image Size:</label>
                <input type="number" id="maxDim" name="maxDim" value="2048" min="512" max="4096" step="128">
            </div>
            <input type="file" id="imageFile" name="image" accept="image/*" required>
            <br>
            <button type="submit">Convert to Paint by Numbers</button>
        </form>
    </div>

    <div class="progress-container" id="progressContainer">
        <div class="progress-bar">
            <div class="progress-fill" id="progressFill" style="width: 0%;">
                <span id="progressPercent">0%</span>
            </div>
        </div>
        <div class="progress-text" id="progressText">Initializing...</div>
    </div>

    <div id="result"></div>

    <script>
        document.getElementById('uploadForm').addEventListener('submit', async (e) => {
            e.preventDefault();

            const formData = new FormData();
            const fileInput = document.getElementById('imageFile');
            const points = document.getElementById('points').value;
            const colors = document.getElementById('colors').value;
            const maxDim = document.getElementById('maxDim').value;

            if (!fileInput.files[0]) {
                alert('Please select an image');
                return;
            }

            formData.append('image', fileInput.files[0]);
            formData.append('points', points);
            formData.append('colors', colors);
            formData.append('maxDimension', maxDim);

            // Show progress bar
            document.getElementById('progressContainer').style.display = 'block';
            document.getElementById('result').innerHTML = '';
            updateProgress(0, 'Starting...');

            // Try SSE first for progress streaming
            try {
                await convertWithSSE(formData);
            } catch (error) {
                document.getElementById('progressContainer').style.display = 'none';
                document.getElementById('result').innerHTML =
                    '<p style="color: red;">Error: ' + error.message + '</p>';
            }
        });

        async function convertWithSSE(formData) {
            return new Promise((resolve, reject) => {
                const xhr = new XMLHttpRequest();

                xhr.open('POST', '/convert', true);
                xhr.setRequestHeader('Accept', 'text/event-stream');

                let buffer = '';
                let lastPosition = 0;

                xhr.onprogress = function() {
                    // Get new data since last read
                    const newData = xhr.responseText.substring(lastPosition);
                    lastPosition = xhr.responseText.length;
                    buffer += newData;

                    // Process complete events (separated by \n\n)
                    const events = buffer.split('\n\n');
                    buffer = events.pop(); // Keep incomplete event in buffer

                    events.forEach(eventText => {
                        if (!eventText.trim()) return;

                        const lines = eventText.split('\n');
                        let eventType = 'message';
                        let data = '';

                        lines.forEach(line => {
                            if (line.startsWith('event:')) {
                                eventType = line.substring(6).trim();
                            } else if (line.startsWith('data:')) {
                                data += line.substring(5).trim();
                            }
                        });

                        if (eventType === 'done') {
                            try {
                                const result = JSON.parse(data);
                                displayResult(result);
                                document.getElementById('progressContainer').style.display = 'none';
                                resolve();
                            } catch (e) {
                                reject(new Error('Failed to parse result: ' + e.message));
                            }
                        } else if (eventType === 'error') {
                            reject(new Error(data));
                        } else if (data) {
                            try {
                                const progress = JSON.parse(data);
                                updateProgress(progress.percent, progress.stage);
                            } catch (e) {
                                console.warn('Failed to parse progress:', e);
                            }
                        }
                    });
                };

                xhr.onerror = function() {
                    reject(new Error('Connection failed'));
                };

                xhr.onload = function() {
                    if (xhr.status !== 200) {
                        reject(new Error('Request failed: ' + xhr.status));
                    }
                };

                xhr.send(formData);
            });
        }

        function updateProgress(percent, stage) {
            document.getElementById('progressFill').style.width = percent + '%';
            document.getElementById('progressPercent').textContent = percent + '%';
            document.getElementById('progressText').textContent = stage;
        }

        function displayResult(data) {
            const imageUrl = 'data:image/png;base64,' + data.image;

            let paletteHtml = '<div class="palette"><h3>Color Palette</h3><div class="color-grid">';
            data.palette.forEach((colorInfo) => {
                paletteHtml += '<div class="color-item">' +
                    '<div class="color-number">#' + colorInfo.number + '</div>' +
                    '<div class="color-swatch" style="background-color: ' + colorInfo.hex + ';"></div>' +
                    '<div class="color-code">' + colorInfo.hex + '</div>' +
                    '<div class="color-cmyk">C:' + colorInfo.c + ' M:' + colorInfo.m + '<br>' +
                    'Y:' + colorInfo.y + ' K:' + colorInfo.k + '</div>' +
                    '</div>';
            });
            paletteHtml += '</div></div>';

            document.getElementById('result').innerHTML =
                '<h2>Result</h2>' +
                '<img src="' + imageUrl + '" alt="Paint by Numbers Result">' +
                '<br><a class="download-btn" href="' + imageUrl + '" download="paint-by-numbers.png">Download Image</a>' +
                paletteHtml;
        }
    </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}
