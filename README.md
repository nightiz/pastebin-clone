# Pastebin Clone

A lightweight, high-performance Pastebin clone built with a Go standard library backend and an Express + Tailwind CSS v4 development UI.

## Architecture

- **Backend**: Go (1.22+), standard library `net/http` only. No web frameworks, no ORMs. Pure-Go SQLite (`modernc.org/sqlite`) enabling static, CGO-free single-binary builds.
- **Frontend**: Express.js server-rendered BFF (Backend-For-Frontend) with Tailwind CSS v4 and EJS templates. Node acts as the consumer to the Go API; no direct browser-to-backend calls.

```
/
├── backend/                  # Go module
│   ├── cmd/server/main.go    # Entrypoint, graceful shutdown
│   ├── internal/
│   │   ├── apierr/           # Standardized JSON error response writer
│   │   ├── config/           # Environment config parsing & validation
│   │   ├── httpserver/       # Mux routing & middleware chain
│   │   └── paste/            # Domain model, SQLite store, service logic, handlers
│   ├── migrations/           # SQLite schema migrations
│   └── Dockerfile            # Multi-stage static distroless build
├── frontend/                 # Express development UI
│   ├── src/input.css         # Tailwind CSS v4 theme entrypoint
│   ├── views/                # EJS server-rendered templates
│   ├── public/               # Static assets & client JS
│   ├── routes/               # Express BFF routes
│   └── lib/apiClient.js      # Backend API fetch client
├── TECHSTACK.md              # Project rules & constraints
├── BACKEND.md                # Backend specifications & endpoints
├── FRONTEND.md               # Frontend UI specifications
└── README.md
```

## Quick Start

### 1. Prerequisites
- **Go**: 1.22 or higher
- **Node.js**: 18 or higher with npm

### 2. Run the Backend
```bash
cd backend
go run ./cmd/server
```
The Go API starts on `http://localhost:8080`.

### 3. Run the Frontend
In another terminal:
```bash
cd frontend
npm install
npm run build:css
npm start
```
The Frontend UI opens on `http://localhost:3000`.

---

## API Endpoints

All endpoints use standard JSON request/response formats except `/raw`.

| Method | Path | Description | Auth / Headers |
|--------|------|-------------|----------------|
| `POST` | `/api/pastes` | Create a new paste | None |
| `GET` | `/api/pastes/{id}` | Fetch paste metadata and content | None |
| `GET` | `/api/pastes/{id}/raw` | Fetch raw paste content (`text/plain`) | None |
| `DELETE` | `/api/pastes/{id}` | Delete paste (requires edit token) | `X-Edit-Token: <token>` |
| `GET` | `/healthz` | Health check endpoint | None |

### Create Paste Request
```json
POST /api/pastes
Content-Type: application/json

{
  "content": "package main\n\nfunc main() {}",
  "title": "main.go",
  "language": "go",
  "expires_in_seconds": 3600,
  "burn_after_read": false
}
```

### Create Paste Response (Status 201)
```json
{
  "id": "7kR2x9Za",
  "title": "main.go",
  "content": "package main\n\nfunc main() {}",
  "language": "go",
  "created_at": "2026-09-24T13:45:00Z",
  "expires_at": "2026-09-24T14:45:00Z",
  "burn_after_read": false,
  "views": 0,
  "edit_token": "a4f89d3..."
}
```
*Note: The `edit_token` is returned only once at creation time and stored hashed (SHA-256) on the server.*

---

## Key Features

- **Base62 ID Generation**: 8-character cryptographically secure IDs using `crypto/rand` with uniform distribution (no modulo bias).
- **Burn After Read**: Atomically deletes the paste on first read inside an immediate SQLite transaction.
- **Dual Expiry Purge**: Lazy check-on-read ensures expired pastes immediately return 404, while a background periodic sweeper reclaims database space.
- **Security & Protection**:
  - Request body limits (`MAX_PASTE_BYTES`, default 512KB).
  - Per-IP write rate limiting (`RATE_LIMIT_PER_MIN`, default 30 write req/min).
  - Constant-time edit token verification (`crypto/subtle`).
  - No paste content logging in structured access logs.
  - Strict input validation at trust boundaries.
- **Zero-CGO Static Binary**: Built using `modernc.org/sqlite`, allowing `CGO_ENABLED=0` static single-binary packaging.

---

## Environment Variables

### Backend (`backend/.env`)
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Port for the backend API |
| `DATABASE_PATH` | `./data/pastes.db` | SQLite database file location |
| `MAX_PASTE_BYTES` | `524288` (512KB) | Maximum allowed paste payload size |
| `RATE_LIMIT_PER_MIN` | `30` | Per-IP write rate limit per minute |
| `DEFAULT_EXPIRY_SECONDS` | `0` (never) | Default expiry if omitted by client |

### Frontend (`frontend/.env`)
| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | Port for the Express UI server |
| `BACKEND_API_URL` | `http://localhost:8080` | URL to the Go backend API |

---

## Running Tests & Verifications

### Backend
```bash
cd backend
go test -v ./...
go vet ./...
go build ./...
```

### Frontend
```bash
cd frontend
npm run build:css
node -e "require('./app')"
```
