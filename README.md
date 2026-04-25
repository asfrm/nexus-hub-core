# Nexus Hub Core v2.0.3

**The Ultimate 3D Printing Farm Orchestration Platform**

A high-performance management system for 3D printer farms with native support for Moonraker/Klipper and Bambu Lab printers (LAN mode).

![Go Version](https://img.shields.io/badge/Go-1.21+-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Platform](https://img.shields.io/badge/Platform-Linux%2FWindows-lightgrey)

---

## 🚀 Quick Start

### Run with Build Script
```bash
./nexus-dev.sh
```

### Manual Build & Run
```bash
go build -o nexus-hub .
./nexus-hub -debug
```

### Access
- **Debug Dashboard:** http://localhost:8080/debug.html
- **API Base:** http://localhost:8080/api
- **WebSocket:** ws://localhost:8080/ws

---

## 📐 System Architecture

Nexus Hub Core operates as a central event-driven hub, coordinating communication between diverse printer vendors and client interfaces.

- **Central Hub:** Pub-Sub engine for real-time event broadcasting.
- **Scheduler:** Automated job assignment with integrated wear-leveling logic.
- **Unified Provider API:** Common interface for Moonraker and Bambu Lab printers.
- **SQLite Storage:** Persistent storage for jobs, history, stats, and spool management.

---

## 🗂️ Project Structure

```
nexus-hub-core/
├── main.go             # Application entry point
├── api/                # Pub-Sub hub and packet definitions
├── db/                 # Database layer (SQLite)
├── dispatcher/         # Manual and auto job dispatching logic
├── moonraker/          # Moonraker-specific client implementation
├── printer/            # Unified printer providers (Bambu, Moonraker)
├── scheduler/          # Wear-balanced job scheduler
├── slicer/             # G-code parsing and slicer integration
├── parser/             # Metadata extraction and file analysis
├── web/                # REST API and WebSocket server
├── gcode_vault/        # Storage for uploaded G-code and thumbnails
├── debug.html          # Built-in orchestration & monitoring panel
├── nexus-dev.sh        # Development automation script
└── README.md           # This file
```

---

## 🎯 Features

### Core Features
- ✅ **Multi-Vendor Support** - Native Moonraker & Bambu Lab integration.
- ✅ **Wear-Leveling Scheduler** - Automatically assigns jobs to the least-worn idle printers.
- ✅ **Slicer Integration** - OctoPrint-compatible API for direct uploads from OrcaSlicer/BambuStudio.
- ✅ **Real-Time Monitoring** - WebSocket-based live status updates and system logs.
- ✅ **Spool Management** - Track filament usage across your farm.
- ✅ **Print Analytics** - Detailed history and performance tracking.

### Bambu Lab Integration
- ✅ **Pure LAN Mode** - Direct MQTT over TLS (port 8883) via `bambusdk-go`.
- ✅ **AMS Support** - Robust mapping for multi-material printing.
- ✅ **FTP Upload** - High-speed file transfer to printer SD card.
- ✅ **Hardware Safety** - G-code execution locks while printer is busy.

### Moonraker/Klipper Integration
- ✅ **REST API** - Full compatibility with Moonraker endpoints.
- ✅ **Direct Print** - Upload and start jobs in one action.

---

## 📖 Usage Guide

### 1. Slicer Configuration
Set your slicer (OrcaSlicer, BambuStudio, etc.) to use the OctoPrint host type:
- **Hostname:** `http://<hub-ip>:8080`
- **API Key:** (None required for local network)

### 2. Printer Setup
Printers are organized into **Clusters** based on their model type (e.g., "Bambu X1C", "Voron 2.4").
1. Create a Cluster via the API/Debug UI.
2. Add printers to the cluster by providing their IP addresses and (for Bambu) access codes.

### 3. Automated Scheduling
Toggle `AutoMode` to allow the scheduler to automatically pick up `PENDING` jobs and assign them to the best available printer in the matching cluster.

---

## 🔌 API Reference (Partial)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/jobs` | GET | List pending jobs |
| `/api/jobs/list` | GET | List active/assigned jobs |
| `/api/jobs/assign` | POST | Manually assign a job |
| `/api/clusters` | GET | List all clusters and their printers |
| `/api/stats` | GET | Summary of system activity |
| `/api/analytics` | GET | Detailed performance metrics |
| `/api/spools` | GET/POST | Manage filament inventory |
| `/api/health` | GET | System health and version info |

---

## 🏗️ Database Schema

```sql
-- Clusters: Groups of identical printer models
CREATE TABLE clusters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    target_model TEXT UNIQUE NOT NULL,
    is_enabled INTEGER DEFAULT 1
);

-- Printers: Physical hardware instances
CREATE TABLE printers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    status TEXT DEFAULT 'IDLE',
    printer_type TEXT DEFAULT 'MOONRAKER',
    serial TEXT,
    access_code TEXT,
    FOREIGN KEY(cluster_id) REFERENCES clusters(id)
);

-- Print Jobs: Queue of work to be performed
CREATE TABLE print_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    filename TEXT NOT NULL,
    filament_type TEXT,
    printer_model TEXT NOT NULL,
    estimated_time_s INTEGER,
    status TEXT NOT NULL, -- PENDING, PRINTING, COMPLETED, etc.
    thumbnail_url TEXT,
    filament_weight_g REAL,
    created_at DATETIME NOT NULL
);

-- Spools: Filament inventory tracking
CREATE TABLE spools (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    material TEXT NOT NULL,
    color TEXT NOT NULL,
    total_weight_g REAL NOT NULL,
    remaining_weight_g REAL NOT NULL,
    assigned_printer_id INTEGER DEFAULT 0
);
```

---

## 📄 License

MIT License - See LICENSE file for details.
