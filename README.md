# Nexus Hub Core

**3D Printing Farm Orchestration System**

A cross-platform (Linux/Windows) management system for 3D printer farms using Moonraker/Klipper.

## Features

- 🖨️ **Multi-Printer Management** - Organize printers into clusters by model
- 📤 **G-Code Upload** - Upload and automatically parse G-Code metadata
- 📋 **Job Queue** - Pending, active, and completed job tracking
- ⚡ **Auto-Scheduling** - Automatic job assignment to available printers
- 🎮 **Printer Control** - Pause, resume, stop, emergency stop
- 📊 **Real-time Dashboard** - Live status updates via WebSocket
- 🔍 **Printer Polling** - Automatic status monitoring every 5 seconds

## Quick Start

### Run the Application

```bash
# Build and run
./nexus-hub-core

# Or with debug logging
./nexus-hub-core -debug
```

### Access the Web UI

Open your browser to: **http://localhost:8080**

### Development Mode (Frontend)

```bash
cd ui
npm install
npm run dev
```

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Dashboard  │     │  Job Queue   │     │  Clusters    │
│   (Stats)    │     │  (Upload)    │     │  (Printers)  │
└──────────────┘     └──────────────┘     └──────────────┘
         │                   │                    │
         └───────────────────┼────────────────────┘
                             │
                    ┌────────▼────────┐
                    │   WebSocket     │
                    │   Pub-Sub Hub   │
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
┌────────▼────────┐ ┌────────▼────────┐ ┌────────▼────────┐
│   Scheduler     │ │   Poller        │ │   Web Server    │
│   (Auto)        │ │   (Status)      │ │   (REST API)    │
└─────────────────┘ └─────────────────┘ └─────────────────┘
                             │
                    ┌────────▼────────┐
                    │    SQLite DB    │
                    └─────────────────┘
                             │
                    ┌────────▼────────┐
                    │   Moonraker     │
                    │   (Printers)    │
                    └─────────────────┘
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed documentation.

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/jobs` | GET | Get pending jobs |
| `/api/jobs/list` | GET | Get active jobs |
| `/api/jobs/assign` | POST | Assign job to printer |
| `/api/clusters` | GET | Get all clusters |
| `/api/printers/add` | POST | Add printer |
| `/api/printers/control` | POST | Control printer |
| `/server/files/upload` | POST | Upload G-Code |
| `/ws` | WS | WebSocket connection |

## Project Structure

```
nexus-hub-core/
├── main.go              # Entry point
├── api/                 # Data structures, Hub
├── db/                  # Database operations
├── web/                 # HTTP server, WebSocket
├── moonraker/           # Printer API client
├── printer/             # Status polling
├── scheduler/           # Job scheduling
├── dispatcher/          # Manual dispatch
├── slicer/              # G-Code parsing
└── ui/                  # React frontend
```

## Requirements

- Go 1.21+
- Node.js 18+ (for frontend development)
- Moonraker/Klipper printers on the network

## License

MIT
