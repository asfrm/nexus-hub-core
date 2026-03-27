# Nexus Hub Core - Architecture & Features

## 📐 System Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NEXUS HUB CORE                                   │
│                     3D Printing Farm Orchestration                       │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                            FRONTEND (React + Vite)                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │  Dashboard  │  │  Job Queue  │  │  Clusters   │  │  Settings   │    │
│  │   (Stats)   │  │  (Upload)   │  │  (Printers) │  │   (Config)  │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
│                           │                  │                           │
│                    WebSocket           REST API                           │
│                   (Real-time)        (HTTP/JSON)                         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            BACKEND (Go)                                  │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      Web Engine (HTTP Server)                    │    │
│  │  - File Upload Handler                                          │    │
│  │  - WebSocket Handler (Hub/Pub-Sub)                              │    │
│  │  - REST API Endpoints                                           │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                    │                                     │
│         ┌──────────────────────────┼──────────────────────────┐         │
│         │                          │                          │         │
│         ▼                          ▼                          ▼         │
│  ┌─────────────┐           ┌─────────────┐            ┌─────────────┐  │
│  │  Scheduler  │           │   Printer   │            │ Dispatcher  │  │
│  │  (Auto-     │           │   Poller    │            │  (Manual)   │  │
│  │  Assign)    │           │  (Status)   │            │             │  │
│  └─────────────┘           └─────────────┘            └─────────────┘  │
│         │                          │                          │         │
│         └──────────────────────────┼──────────────────────────┘         │
│                                    │                                     │
│                                    ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                    Event Hub (Pub-Sub Pattern)                   │    │
│  │  - Broadcasts events to all subscribers                         │    │
│  │  - WebSocket clients                                            │    │
│  │  - Internal modules                                             │    │
│  └─────────────────────────────────────────────────────────────────┘    │
│                                    │                                     │
│                                    ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────┐    │
│  │                      Database (SQLite)                           │    │
│  │  - system_logs                                                  │    │
│  │  - print_jobs                                                   │    │
│  │  - clusters                                                     │    │
│  │  - printers                                                     │    │
│  └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      MOONRAKER / KLIPPER API                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │  Printer 1  │  │  Printer 2  │  │  Printer 3  │  │  Printer N  │    │
│  │  (Vorons)   │  │  (Voros)    │  │  (BOMs)     │  │  (...)      │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 🗂️ Module Structure

```
nexus-hub-core/
├── main.go                 # Application entry point, orchestration
├── api/
│   └── protocol.go         # NexusPacket, Hub (Pub-Sub), data structures
├── db/
│   └── database.go         # SQLite operations, connection pooling
├── web/
│   ├── server.go           # HTTP server, REST API routes
│   └── panel.go            # WebSocket handler with heartbeat
├── moonraker/
│   ├── client.go           # File upload, printer commands
│   └── status.go           # Printer status polling, info retrieval
├── printer/
│   └── poller.go           # Periodic printer status checks
├── scheduler/
│   └── scheduler.go        # Automatic job assignment logic
├── dispatcher/
│   └── manager.go          # Manual dispatch support
├── slicer/
│   ├── emulator.go         # G-Code upload handler
│   └── parser.go           # G-Code metadata extraction
└── ui/                     # React frontend (built assets)
```

---

## 🔄 Data Flow

### Job Submission Flow
```
1. User uploads G-Code → /server/files/upload
2. Parser extracts metadata (model, filament, time)
3. Job created in DB (PENDING status)
4. System log broadcast via WebSocket
5. Frontend updates job queue in real-time
```

### Automatic Assignment Flow
```
1. Scheduler runs every 10 seconds
2. Checks AutoMode flag
3. Finds PENDING jobs
4. Matches job printer_model to cluster target_model
5. Finds IDLE printer in matching cluster
6. Assigns job → Uploads to printer → Starts print
7. Updates job status to PRINTING
8. Broadcasts status update
```

### Printer Status Polling Flow
```
1. Poller runs every 5 seconds
2. Queries each enabled cluster's printers
3. Calls Moonraker API: /printer/objects/query
4. Parses status (printing, paused, complete, error)
5. Updates DB if status changed
6. Broadcasts PRINTER_STATUS event
7. Frontend updates printer cards in real-time
```

---

## ✨ Key Features

### Current Features
| Feature | Status | Description |
|---------|--------|-------------|
| G-Code Upload | ✅ | Upload and parse G-Code files |
| Job Queue | ✅ | Pending/Active/Completed job management |
| Cluster Management | ✅ | Group printers by model |
| Printer Control | ✅ | Pause, Resume, Stop, Emergency Stop |
| Auto-Assignment | ✅ | Automatic job scheduling |
| Manual Assignment | ✅ | Manual job-to-printer assignment |
| Real-time Updates | ✅ | WebSocket-based live updates |
| Printer Polling | ✅ | Status monitoring every 5s |
| System Logs | ✅ | Centralized logging with levels |
| Cross-Platform | ✅ | Windows + Linux support |

### Suggested New Features

#### 🔥 High Priority
1. **Print Progress Dashboard**
   - Real-time progress bars for all active prints
   - Time remaining estimates
   - Layer counter display
   - Temperature graphs

2. **Filament Management**
   - Track filament usage per job
   - Low filament alerts
   - Filament inventory database
   - Automatic filament change reminders

3. **Print History & Analytics**
   - Successful/failed print tracking
   - Print time analytics
   - Failure rate per printer/model
   - Export reports (CSV/PDF)

4. **Multi-Hub Clustering**
   - Multiple Nexus Hub instances
   - Central coordinator for large farms
   - Load balancing across hubs

#### 📊 Medium Priority
5. **User Authentication**
   - Role-based access (Admin, Operator, Viewer)
   - API key management
   - Audit logs

6. **Webhook Integrations**
   - Discord/Slack notifications
   - Print completion webhooks
   - Error alerts to messaging platforms

7. **G-Code Preview**
   - 3D G-Code visualization
   - Layer-by-layer preview
   - Estimated filament usage display

8. **Printer Calibration Profiles**
   - Store printer-specific settings
   - Auto-apply profiles on job assignment
   - Bed leveling history

#### 🎯 Nice to Have
9. **Mobile App**
   - React Native version
   - Push notifications
   - QR code job submission

10. **Backup & Restore**
    - Database backup scheduling
    - Configuration export/import
    - Cloud sync option

11. **Plugin System**
    - Custom script execution
    - Third-party integrations
    - Webhook builder

12. **AI-Powered Features**
    - Print failure prediction
    - Optimal print scheduling
    - Anomaly detection

---

## 🔌 API Reference

### REST Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/jobs` | Get pending jobs |
| GET | `/api/jobs/list` | Get active jobs |
| GET | `/api/jobs/completed` | Get completed jobs |
| POST | `/api/jobs/assign` | Assign job to printer |
| GET | `/api/clusters` | Get all clusters |
| POST | `/api/clusters/create` | Create new cluster |
| DELETE | `/api/clusters/delete?id=` | Delete cluster |
| POST | `/api/clusters/toggle?id=&enabled=` | Toggle cluster |
| POST | `/api/printers/add` | Add printer to cluster |
| GET | `/api/printers/status?id=` | Get printer status |
| POST | `/api/printers/control?id=&action=` | Control printer |
| POST | `/server/files/upload` | Upload G-Code file |
| GET | `/api/setup/unassigned` | Get unassigned models |
| POST | `/api/setup/analyze-gcode` | Analyze G-Code file |
| GET | `/api/stats` | Get system statistics |
| WS | `/ws` | WebSocket connection |

### WebSocket Events

```typescript
// System Log Event
{
  event_type: "SYSTEM_LOG",
  log: {
    module: "SLICER" | "WEB" | "POLLER" | "SCHEDULER",
    level: "INFO" | "ERROR" | "WARN",
    message: string,
    timestamp: ISO8601
  }
}

// Printer Status Event
{
  event_type: "PRINTER_STATUS",
  payload: {
    printer_id: number,
    status: "IDLE" | "PRINTING" | "PAUSED" | "ERROR",
    filename: string,
    progress: number,
    time_left: number,
    temp_nozzle: number,
    temp_bed: number
  }
}
```

---

## 🚀 Quick Start

### Backend
```bash
cd /home/asfrm/nexus-hub-core
go build -o nexus-hub-core .
./nexus-hub-core              # Normal mode
./nexus-hub-core -debug       # Debug logging
```

### Frontend (Development)
```bash
cd /home/asfrm/nexus-hub-core/ui
npm install
npm run dev
```

### Frontend (Production Build)
```bash
cd /home/asfrm/nexus-hub-core/ui
npm run build
cp -r dist/* ../ui/
```

---

## 📊 Database Schema

```sql
-- System Logs
CREATE TABLE system_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    module TEXT NOT NULL,
    level TEXT NOT NULL,
    message TEXT NOT NULL
);

-- Printer Clusters
CREATE TABLE clusters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    target_model TEXT UNIQUE NOT NULL,
    is_enabled INTEGER DEFAULT 1
);

-- Printers
CREATE TABLE printers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    status TEXT DEFAULT 'IDLE',
    FOREIGN KEY(cluster_id) REFERENCES clusters(id)
);

-- Print Jobs
CREATE TABLE print_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    filament_type TEXT,
    printer_model TEXT NOT NULL,
    estimated_time_s INTEGER,
    status TEXT NOT NULL,
    target_cluster_id INTEGER DEFAULT 0,
    target_printer_id INTEGER DEFAULT 0,
    created_at DATETIME NOT NULL
);
```

---

## 🔧 Configuration

### Environment Variables (Future)
```bash
NEXUS_DB_PATH=./hub.db
NEXUS_PORT=8080
NEXUS_POLL_INTERVAL=5s
NEXUS_SCHEDULER_INTERVAL=10s
NEXUS_AUTO_MODE=false
```

### Command Line Flags
```bash
-debug    Enable debug logging
```

---

## 📝 Future Improvements

1. **Add Prometheus metrics endpoint** for monitoring
2. **Implement health check endpoint** for load balancers
3. **Add configuration file support** (YAML/TOML)
4. **Implement job priorities** in scheduler
5. **Add printer maintenance scheduling**
6. **Create mobile-responsive UI**
7. **Add dark/light theme toggle**
8. **Implement job templates** for recurring prints
9. **Add batch operations** for multiple printers
10. **Create API documentation** with Swagger/OpenAPI
