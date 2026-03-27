# 🚀 Nexus Hub Core - Quick Start Guide

## Running the Application

### Option 1: Automated Build & Run Script (Recommended)

Use the included `nexus-dev.sh` script that automates the entire development lifecycle:

```bash
cd /home/asfrm/nexus-hub-core
./nexus-dev.sh
```

**What the script does:**
1. ✅ Kills any existing `nexus-hub-core` processes
2. ✅ Removes old build artifacts (`ui/dist/`, `dist/`)
3. ✅ Installs npm dependencies (if needed)
4. ✅ Builds frontend (React + Vite)
5. ✅ Deploys assets to root `dist/` folder
6. ✅ Runs `go mod tidy`
7. ✅ Builds Go backend
8. ✅ Starts the application with debug logging

**Access the application:**
- **Web UI:** http://localhost:8080
- **API:** http://localhost:8080/api
- **WebSocket:** ws://localhost:8080/ws

---

### Option 2: Manual Build & Run

#### Step 1: Build Frontend
```bash
cd ui
npm install
npm run build
cd ..
```

#### Step 2: Deploy Assets
```bash
# Create root dist folder and copy built assets
rm -rf dist
mkdir -p dist
cp -r ui/dist/* dist/
```

#### Step 3: Build Backend
```bash
go mod tidy
go build -o nexus-hub-core
```

#### Step 4: Run Application
```bash
./nexus-hub-core -debug
```

---

### Option 3: Run Pre-Built Binary

If you already have a built binary:

```bash
./nexus-hub-core -debug
```

---

## 📋 Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| **Go** | 1.21+ | Backend runtime |
| **Node.js** | 18+ | Frontend build |
| **npm** | 9+ | Package manager |

### Check Installed Versions

```bash
go version      # Should show go1.21 or higher
node --version  # Should show v18 or higher
npm --version   # Should show v9 or higher
```

---

## 🔧 Configuration

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-debug` | Enable debug logging | false |

### Environment Variables (Future)

| Variable | Description | Default |
|----------|-------------|---------|
| `NEXUS_PORT` | HTTP server port | 8080 |
| `NEXUS_DB_PATH` | SQLite database path | hub.db |

---

## 📁 Project Structure

```
nexus-hub-core/
├── nexus-dev.sh          # Build & run automation script
├── main.go               # Go entry point
├── nexus-hub-core        # Compiled binary (after build)
├── hub.db                # SQLite database (created on first run)
├── dist/                 # Frontend assets (created by build)
│   ├── index.html
│   ├── assets/
│   │   ├── index-*.css
│   │   └── index-*.js
├── ui/                   # Frontend source
│   ├── src/
│   ├── package.json
│   └── dist/             # Built frontend (temporary)
├── api/                  # API definitions
├── db/                   # Database layer
├── web/                  # Web server
├── printer/              # Printer providers (Moonraker, BambuLab)
└── scheduler/            # Job scheduling
```

---

## 🎨 First-Time Setup

### 1. Create Your First Cluster

1. Open http://localhost:8080
2. Navigate to **Printers** tab
3. Click **+ Create Cluster**
4. Enter:
   - **Cluster Name:** e.g., "Production Line A"
   - **Target Model:** e.g., "Voron 2.4" or "Bambu Lab X1C"
5. Click **Create**

### 2. Add a Printer

#### For Moonraker/Klipper Printers:
1. Click **+ Add Printer** on your cluster
2. Select **Moonraker / Klipper**
3. Enter:
   - **Printer Name:** e.g., "Printer 1"
   - **IP Address:** e.g., "192.168.1.100"
4. Click **Add Printer**

#### For Bambu Lab Printers:

**LAN Mode (Direct Connection):**
1. Click **+ Add Printer** on your cluster
2. Select **Bambu Lab (Cloud/Local)**
3. Toggle to **🏠 LAN (Local)**
4. Enter:
   - **Printer IP Address:** e.g., "192.168.1.100"
   - **Access Code:** 6-character code from printer screen
5. Click **Add LAN Printer**

**Cloud Mode (Bambu Cloud):**
1. Click **+ Add Printer** on your cluster
2. Select **Bambu Lab (Cloud/Local)**
3. Toggle to **☁️ Cloud (Bambu)**
4. Click **🔐 Sign in to Bambu Lab**
5. Enter:
   - **Email:** Your Bambu Lab account email
   - **Password:** Your Bambu Lab password
   - **Region:** North America / Europe / China
6. If 2FA enabled, enter the 6-digit code
7. Click **🖨️ Discover Printers**
8. Select your printer from the list
9. Choose the target cluster
10. Click **Bind Printer**

### 3. Upload a G-Code File

1. Navigate to **Job Queue** tab
2. Click **📁 Upload G-Code**
3. Select your `.gcode` file
4. The system will automatically parse:
   - Printer model
   - Filament type
   - Estimated print time

### 4. Assign a Job

1. Find your uploaded job in **Pending Jobs**
2. Click **Assign**
3. Select the target cluster
4. Select an available printer
5. Click **Assign & Start**

---

## 🔍 Debugging

### View Logs

The application logs to console with the following prefixes:

- `[API DEBUG]` - API endpoint calls
- `[BAMBU]` - Bambu Lab authentication and operations
- `[POLLER]` - Printer status polling
- `[SCHEDULER]` - Job scheduling events
- `[WEB]` - Web server events
- `[SLICER]` - G-Code processing

### Enable Debug Mode

```bash
./nexus-hub-core -debug
```

### Common Issues

**Frontend not loading:**
```bash
# Rebuild frontend
cd ui
npm run build
cd ..
rm -rf dist
mkdir dist
cp -r ui/dist/* dist/
```

**Port already in use:**
```bash
# Kill existing process
pkill -f nexus-hub-core
# Or use a different port (modify main.go)
```

**Bambu Lab authentication fails:**
- Check your email/password
- Verify region selection (US/EU/CN)
- If 2FA enabled, ensure you enter the code within 30 seconds
- Check console logs for `[API DEBUG] Bambu discover: Authentication failed`

---

## 🛑 Stopping the Application

Press **Ctrl+C** in the terminal to gracefully shut down.

---

## 📊 API Endpoints

### Jobs
- `GET /api/jobs` - Get pending jobs
- `GET /api/jobs/list` - Get active jobs
- `POST /api/jobs/assign` - Assign job to printer

### Clusters & Printers
- `GET /api/clusters` - Get all clusters
- `POST /api/clusters/create` - Create cluster
- `DELETE /api/clusters/delete?id=` - Delete cluster
- `POST /api/printers/add` - Add printer

### Bambu Lab Cloud
- `POST /api/bambu/login` - Save Bambu account
- `POST /api/bambu/verify-2fa` - Verify 2FA code
- `GET /api/bambu/discover` - Discover printers
- `POST /api/bambu/bind` - Bind printer to cluster

### Analytics
- `GET /api/analytics` - System analytics
- `GET /api/history` - Print history
- `GET /api/printers/wear` - Wear-level statistics

---

## 🔄 Updating

To update to the latest version:

```bash
# Pull latest changes (if using git)
git pull

# Rebuild everything
./nexus-dev.sh
```

---

## 📞 Support

- **Documentation:** See `ARCHITECTURE.md`, `ROADMAP.md`, `LAUNCH.md`
- **Issues:** GitHub Issues
- **Community:** Discord server (coming soon)

---

**Ready to orchestrate your 3D printing farm!** 🎉
