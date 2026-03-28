# Nexus Hub Core v2.0

**The Ultimate 3D Printing Farm Orchestration Platform**

A cross-platform (Linux/Windows) management system for 3D printer farms with native support for Moonraker/Klipper and Bambu Lab printers via MQTT.

![Go Version](https://img.shields.io/badge/Go-1.21+-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Platform](https://img.shields.io/badge/Platform-Linux%2FWindows-lightgrey)

---

## 🚀 Quick Start

### Run with Build Script
```bash
./nexus-dev.sh
```

### Manual Build
```bash
cd cmd/nexus-hub
go build -o nexus-hub .
./nexus-hub -debug
```

### Access
- **Web UI:** http://localhost:8080
- **API:** http://localhost:8080/api
- **WebSocket:** ws://localhost:8080/ws

---

## 📐 System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        NEXUS HUB CORE                            │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    Frontend (React)                       │   │
│  │  Dashboard │ Job Queue │ Clusters │ Analytics │ Settings  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                    WebSocket │ REST API                          │
│                              │                                   │
│  ┌───────────────────────────┴──────────────────────────────┐   │
│  │                   Backend (Go)                            │   │
│  │                                                           │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │   │
│  │  │  Web Server │  │  Printer    │  │   Network       │   │   │
│  │  │  (REST/WS)  │  │  Providers  │  │   Scanner       │   │   │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │   │
│  │         │                │                                 │   │
│  │  ┌──────┴────────────────┴──────┐                         │   │
│  │  │      Event Hub (Pub-Sub)     │                         │   │
│  │  └──────────────┬───────────────┘                         │   │
│  │                 │                                          │   │
│  │  ┌──────────────┴───────────────┐                         │   │
│  │  │       SQLite Database         │                         │   │
│  │  └───────────────────────────────┘                         │   │
│  └───────────────────────────────────────────────────────────┘   │
│                              │                                    │
└──────────────────────────────┼────────────────────────────────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
   ┌─────▼─────┐        ┌─────▼─────┐        ┌─────▼─────┐
   │ Moonraker │        │  Bambu    │        │  Bambu    │
   │  Klipper  │        │   LAN     │        │  Cloud    │
   │ :7125     │        │  :8883    │        │  :8883    │
   └───────────┘        └───────────┘        └───────────┘
```

---

## 🗂️ Project Structure

```
nexus-hub-core/
├── cmd/
│   └── nexus-hub/          # Application entry point
│       └── main.go
├── internal/
│   ├── api/                # REST API & WebSocket handlers
│   │   ├── hub.go          # Pub-Sub event hub
│   │   ├── server.go       # HTTP server & routes
│   │   └── panel.go        # WebSocket handler
│   ├── db/                 # Database layer
│   │   └── database.go     # SQLite operations
│   ├── printer/            # Printer providers
│   │   ├── poller.go       # Status polling engine
│   │   └── bambu/          # Bambu Lab MQTT client
│   │       └── client.go   # Native MQTT implementation
│   ├── scanner/            # Network discovery
│   │   └── scanner.go      # SSDP/UDP printer discovery
│   └── parser/             # File parsing
│       ├── emulator.go     # G-Code upload handler
│       └── parser.go       # G-Code metadata extraction
├── ui/                     # React frontend (built assets)
├── go.mod                  # Go module definition
├── go.sum                  # Go dependencies
├── nexus-dev.sh            # Build & run automation
└── README.md               # This file
```

---

## 🎯 Features

### Core Features
- ✅ **Multi-Vendor Support** - Moonraker/Klipper & Bambu Lab
- ✅ **Native MQTT** - No plugins, pure Go implementation
- ✅ **Auto-Discovery** - SSDP/UDP network scanning
- ✅ **Real-Time Updates** - WebSocket live status
- ✅ **Job Queue** - Pending/Active/Completed tracking
- ✅ **Cluster Management** - Group printers by model
- ✅ **Wear-Leveling** - Balanced job assignment
- ✅ **Print Analytics** - History, stats, wear tracking
- ✅ **Cross-Platform** - Linux & Windows support

### Bambu Lab Integration
- ✅ **LAN Mode** - Direct MQTT over TLS (port 8883)
- ✅ **Cloud Mode** - Bambu cloud MQTT brokers
- ✅ **8-Char Access Code** - Secure local authentication
- ✅ **Real-Time Status** - Temperatures, progress, layers
- ✅ **Full Control** - G-code, pause, resume, stop

### Moonraker/Klipper Integration
- ✅ **HTTP API** - Full Moonraker compatibility
- ✅ **WebSocket** - Real-time status updates
- ✅ **File Upload** - Direct G-Code upload & print
- ✅ **Command Control** - Pause, resume, cancel, estop

---

## 🔧 Installation

### Prerequisites
- Go 1.21+
- Node.js 18+ (for frontend development)
- SQLite (embedded, no installation needed)

### Build from Source

1. **Clone Repository**
```bash
git clone https://github.com/yourusername/nexus-hub-core.git
cd nexus-hub-core
```

2. **Build Backend**
```bash
cd cmd/nexus-hub
go build -o nexus-hub .
```

3. **Build Frontend** (optional, pre-built included)
```bash
cd ../../ui
npm install
npm run build
```

4. **Run**
```bash
./nexus-hub -debug
```

### Using Build Script
```bash
chmod +x nexus-dev.sh
./nexus-dev.sh
```

---

## 📖 Usage Guide

### First-Time Setup

#### 1. Create a Cluster
1. Navigate to **Printers** tab
2. Click **+ Create Cluster**
3. Enter cluster name and target printer model
4. Click **Create**

#### 2. Add a Printer

**For Moonraker/Klipper:**
1. Click **+ Add Printer** on your cluster
2. Select **Moonraker / Klipper**
3. Enter printer name and IP address
4. Click **Add Printer**

**For Bambu Lab (LAN):**
1. Click **+ Add Printer** on your cluster
2. Select **Bambu Lab (Cloud/Local)**
3. Toggle to **🏠 LAN (Local)**
4. Enter printer IP and 8-character access code
5. Click **Add LAN Printer**

**For Bambu Lab (Cloud):**
1. Click **+ Add Printer** on your cluster
2. Select **Bambu Lab (Cloud/Local)**
3. Toggle to **☁️ Cloud (Bambu)**
4. Click **🌐 Sign in via Bambu Lab**
5. Enter token from Bambu profile
6. Discover and bind printers

#### 3. Upload a Print Job
1. Navigate to **Job Queue** tab
2. Click **📁 Upload G-Code**
3. Select `.gcode` or `.3mf` file
4. System auto-parses metadata

#### 4. Assign a Job
1. Find job in **Pending Jobs**
2. Click **Assign**
3. Select cluster and printer
4. Click **Assign & Start**

---

## 🔌 API Reference

### REST Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/jobs` | GET | Get pending jobs |
| `/api/jobs/list` | GET | Get active jobs |
| `/api/jobs/completed` | GET | Get completed jobs |
| `/api/jobs/assign` | POST | Assign job to printer |
| `/api/clusters` | GET | Get all clusters |
| `/api/clusters/create` | POST | Create cluster |
| `/api/clusters/delete` | DELETE | Delete cluster |
| `/api/printers/add` | POST | Add printer |
| `/api/printers/control` | POST | Control printer |
| `/api/analytics` | GET | System analytics |
| `/api/history` | GET | Print history |
| `/api/bambu/login-url` | GET | Get Bambu login URL |
| `/api/bambu/token` | POST | Save Bambu token |
| `/api/bambu/discover` | GET | Discover Bambu printers |
| `/api/scanner/discover` | GET | Network printer scan |

### WebSocket Events

**System Log:**
```json
{
  "event_type": "SYSTEM_LOG",
  "log": {
    "module": "SLICER",
    "level": "INFO",
    "message": "Task uploaded successfully",
    "timestamp": "2026-03-28T12:00:00Z"
  }
}
```

**Printer Status:**
```json
{
  "event_type": "PRINTER_STATUS",
  "payload": {
    "printer_id": 1,
    "status": "PRINTING",
    "progress": 45.5,
    "time_left": 3600,
    "temperatures": {
      "nozzle": 200.5,
      "bed": 60.0
    }
  }
}
```

---

## 🏗️ Database Schema

```sql
-- System Logs
CREATE TABLE system_logs (
    id INTEGER PRIMARY KEY,
    timestamp DATETIME,
    module TEXT,
    level TEXT,
    message TEXT
);

-- Clusters
CREATE TABLE clusters (
    id INTEGER PRIMARY KEY,
    name TEXT,
    target_model TEXT UNIQUE,
    is_enabled INTEGER DEFAULT 1
);

-- Printers
CREATE TABLE printers (
    id INTEGER PRIMARY KEY,
    cluster_id INTEGER,
    name TEXT,
    ip_address TEXT,
    status TEXT DEFAULT 'IDLE',
    printer_type TEXT DEFAULT 'MOONRAKER',
    FOREIGN KEY(cluster_id) REFERENCES clusters(id)
);

-- Print Jobs
CREATE TABLE print_jobs (
    id INTEGER PRIMARY KEY,
    filename TEXT,
    filament_type TEXT,
    printer_model TEXT,
    estimated_time_s INTEGER,
    status TEXT,
    target_cluster_id INTEGER,
    target_printer_id INTEGER,
    created_at DATETIME,
    started_at DATETIME,
    completed_at DATETIME
);

-- Print History
CREATE TABLE print_history (
    id INTEGER PRIMARY KEY,
    job_id INTEGER,
    filename TEXT,
    printer_id INTEGER,
    actual_time_s INTEGER,
    status TEXT,
    completed_at DATETIME
);

-- Printer Stats (Wear Tracking)
CREATE TABLE printer_stats (
    printer_id INTEGER PRIMARY KEY,
    total_print_time_s INTEGER,
    total_prints INTEGER,
    successful_prints INTEGER,
    failed_prints INTEGER
);
```

---

## 🔐 Security Notes

### Bambu Lab LAN Mode
- Uses TLS with self-signed certificates
- `InsecureSkipVerify: true` required
- 8-character access code from printer screen
- All communication encrypted

### Bambu Lab Cloud Mode
- Uses official Bambu MQTT brokers
- Token-based authentication
- Region-specific brokers (US/EU/CN)

### Recommendations
- Run on isolated network segment
- Use firewall rules to restrict access
- Enable authentication in production
- Regular database backups

---

## 🐛 Troubleshooting

### Bambu Printer Not Connecting
1. Verify 8-character access code (Settings → About on printer)
2. Check firewall allows port 8883
3. Ensure printer and hub on same network (for LAN mode)
4. Check logs: `[BAMBU MQTT] Connection lost: ...`

### Frontend Not Loading
```bash
# Rebuild frontend
cd ui
npm run build
cp -r dist/* ../ui/
```

### Database Locked
```bash
# Kill existing processes
pkill -f nexus-hub
# Remove lock file
rm hub.db-shm hub.db-wal
```

---

## 📊 Performance

| Metric | Value |
|--------|-------|
| Max Printers | Unlimited |
| Polling Interval | 5 seconds |
| WebSocket Latency | <100ms |
| Memory Usage | ~50MB |
| Binary Size | ~15MB |

---

## 🤝 Contributing

1. Fork repository
2. Create feature branch
3. Make changes
4. Run tests
5. Submit PR

### Development Setup
```bash
# Backend
cd cmd/nexus-hub
go run main.go -debug

# Frontend
cd ui
npm run dev
```

---

## 📄 License

MIT License - See LICENSE file for details.

---

## 🙏 Credits

Built with:
- Go (paho.mqtt.golang, gorilla/websocket, modernc.org/sqlite)
- React (Vite, Tailwind CSS)
- Moonraker/Klipper API
- Bambu Lab MQTT Protocol

---

**Ready to orchestrate your 3D printing farm!** 🎉
