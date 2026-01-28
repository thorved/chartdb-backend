FROM golang:1.25.6-alpine AS backend-builder

WORKDIR /app

# Copy Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (pure Go, no CGO needed)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /chartdb-backend

# Sync Dashboard (Vue) builder
FROM node:18-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend files
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# ChartDB (React) builder - main application
# ChartDB requires Node 24 as per .nvmrc
FROM node:24-alpine AS chartdb-builder

WORKDIR /app/chartdb

# Copy ChartDB submodule files
COPY chartdb/package*.json ./
RUN npm install

COPY chartdb/ .

# Build ChartDB for production (skip lint, just compile TypeScript and build)
RUN npx tsc -b && npx vite build

# Inject sync toolbar script into the built index.html
RUN sed -i 's|</body>|<script src="/static/sync-toolbar.js"></script></body>|' /app/chartdb/dist/index.html

# Final image - single unified image
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite

# Copy Go backend binary
COPY --from=backend-builder /chartdb-backend .

# Copy Sync Dashboard (Vue) to /frontend/dist - served at /sync/
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Copy ChartDB (React) to /chartdb/dist - served at /
COPY --from=chartdb-builder /app/chartdb/dist ./chartdb/dist

# Copy static assets for sync toolbar
COPY static/ ./static/

# Create data directory for SQLite
RUN mkdir -p /app/data

# Environment variables
ENV PORT=8080
ENV GIN_MODE=release
ENV DATABASE_PATH=/app/data/chartdb_sync.db

EXPOSE 8080

CMD ["./chartdb-backend"]
