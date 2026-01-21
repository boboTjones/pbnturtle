# Stage 1: Build WASM binary
FROM golang:alpine AS builder

WORKDIR /build

# Copy Go source files
COPY wasm/ ./wasm/

# Build WASM binary
WORKDIR /build/wasm
RUN GOOS=js GOARCH=wasm go build -o ../paintbynumbers.wasm

# Stage 2: Serve static files
FROM nginx:alpine

# Remove default nginx config
RUN rm -rf /usr/share/nginx/html/*

# Copy static files
COPY index.html /usr/share/nginx/html/
COPY worker.js /usr/share/nginx/html/
COPY wasm_exec.js /usr/share/nginx/html/

# Copy WASM binary from builder stage
COPY --from=builder /build/paintbynumbers.wasm /usr/share/nginx/html/

# Configure nginx to listen on port 8080 and proper MIME types
RUN echo 'server { \
    listen 8080; \
    listen [::]:8080; \
    server_name _; \
    root /usr/share/nginx/html; \
    index index.html; \
    location / { \
        try_files $uri $uri/ =404; \
    } \
    types { \
        text/html html htm; \
        text/css css; \
        application/javascript js; \
        application/wasm wasm; \
    } \
}' > /etc/nginx/conf.d/default.conf

# Expose port 8080 for fly.io
EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
