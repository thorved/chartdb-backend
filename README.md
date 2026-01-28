# ChartDB Backend Sync

A Go backend service for syncing ChartDB diagrams stored in browser IndexedDB with a SQLite database. Provides user authentication, versioned push/pull operations, and a web dashboard for managing synced diagrams.

## Features

- 🔐 **User Authentication** - Signup, login with JWT tokens
- 📤 **Push Diagrams** - Sync browser data to server with versioning
- 📥 **Pull Diagrams** - Retrieve diagrams from server to browser
- 📜 **Version History** - Keep track of diagram changes (last 10 versions)
- 🗑️ **Delete Diagrams** - Remove synced diagrams
- 🎨 **Vue Dashboard** - Modern UI at `/sync/` for managing diagrams

## Architecture

```
/sync/                    → Vue SPA Dashboard
/sync/api/auth/*          → Authentication endpoints
/sync/api/diagrams/*      → Diagram sync endpoints
/*                        → Proxy to ChartDB (optional)
```

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js 18+ (for frontend build)

### Backend Setup

```bash
# Clone the repository
git clone https://github.com/thorved/chartdb-backend.git
cd chartdb-backend

# Copy environment file
cp example.env .env

# Download Go dependencies
go mod tidy

# Run the server
go run main.go
```

### Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Build for production
npm run build

# Or run in development mode
npm run dev
```

### Running in Production

```bash
# Build the frontend first
cd frontend && npm run build && cd ..

# Set production environment
export GIN_MODE=release
export JWT_SECRET=your-secure-secret-key
export PORT=8080

# Run the server
go run main.go
```

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/sync/api/auth/signup` | Register new user |
| POST | `/sync/api/auth/login` | Login and get JWT token |
| GET | `/sync/api/auth/me` | Get current user (auth required) |
| PUT | `/sync/api/auth/me` | Update user profile (auth required) |
| PUT | `/sync/api/auth/password` | Change password (auth required) |

### Diagrams

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/sync/api/diagrams` | List all user diagrams |
| GET | `/sync/api/diagrams/:id` | Get diagram info |
| POST | `/sync/api/diagrams/push` | Push diagram from browser |
| GET | `/sync/api/diagrams/pull/:id` | Pull diagram to browser |
| GET | `/sync/api/diagrams/pull/:id?version=N` | Pull specific version |
| DELETE | `/sync/api/diagrams/:id` | Delete diagram |
| GET | `/sync/api/diagrams/:id/versions` | Get version history |

## Usage with ChartDB

### Push (Browser → Server)

1. Open ChartDB in your browser
2. Export your diagram as JSON (File → Export → JSON)
3. Go to `/sync/dashboard`
4. Click "Push from Browser"
5. Paste the JSON and click Push

### Pull (Server → Browser)

1. Go to `/sync/dashboard`
2. Find your diagram and click "Pull"
3. Copy the JSON from the modal
4. In ChartDB, import the JSON (File → Import)

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `GIN_MODE` | Gin mode (`debug`/`release`) | `debug` |
| `JWT_SECRET` | Secret key for JWT tokens | (dev default) |
| `DATABASE_PATH` | SQLite database file path | `chartdb_sync.db` |
| `CHARTDB_URL` | ChartDB frontend URL for proxying | (disabled) |

## Data Schema

The backend stores the complete ChartDB diagram structure:

- **Diagrams** - Main diagram metadata
- **Tables** - Database tables with fields and indexes
- **Relationships** - Foreign key relationships
- **Dependencies** - Table dependencies (views)
- **Areas** - Visual grouping areas
- **Notes** - Diagram notes
- **Custom Types** - Enums and composite types
- **Versions** - Full JSON snapshots for version history

## Docker Deployment

### Build and Run

```bash
docker build -t chartdb-backend .
docker run -p 8080:8080 -v chartdb-data:/app/data chartdb-backend
```

### Docker Compose

```bash
docker-compose up -d
```

## Security Notes

- Always change `JWT_SECRET` in production
- Use HTTPS in production
- Consider rate limiting for auth endpoints
- Database file should be backed up regularly

## License

MIT License