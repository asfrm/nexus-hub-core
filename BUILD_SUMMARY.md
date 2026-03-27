# ✅ Build & Run Utility - Implementation Complete

## 📜 Created: `nexus-dev.sh`

A comprehensive bash script that automates the entire development lifecycle.

### Features

1. **Cleanup Phase**
   - Kills existing `nexus-hub-core` processes
   - Removes `ui/dist/` and root `dist/` folders
   - Removes old binary

2. **Frontend Build**
   - Navigates to `ui/` directory
   - Runs `npm install` (if needed)
   - Executes `npm run build`

3. **Asset Deployment**
   - Creates root-level `dist/` folder
   - Copies all assets from `ui/dist/` to `dist/`

4. **Backend Preparation**
   - Runs `go mod tidy`

5. **Binary Compilation**
   - Builds Go backend: `go build -o nexus-hub-core`

6. **Execution**
   - Starts application: `./nexus-hub-core -debug`

### Visual Output

```
╔════════════════════════════════════════════════════════╗
║   Nexus Hub Core - Development Build & Run Utility    ║
╚════════════════════════════════════════════════════════╝

[1/6] Cleaning up old processes and artifacts...
   ✓ Cleanup complete

[2/6] Building frontend (React + Vite)...
   ✓ Frontend build complete

[3/6] Deploying assets to root dist/ folder...
   ✓ Assets deployed to dist/

[4/6] Preparing Go backend...
   ✓ Go modules updated

[5/6] Building Go backend...
   ✓ Backend build complete

[6/6] Starting Nexus Hub Core...

╔════════════════════════════════════════════════════════╗
║  Build successful! Starting application...             ║
║                                                        ║
║  Web UI:    http://localhost:8080                      ║
║  API:       http://localhost:8080/api                  ║
║  WebSocket: ws://localhost:8080/ws                     ║
║                                                        ║
║  Press Ctrl+C to stop                                  ║
╚════════════════════════════════════════════════════════╝
```

---

## 🔧 Backend Updates

### 1. Static File Path Changed

**File:** `web/server.go`

```go
// OLD: Served from ui/
mux.Handle("/", serveWithCacheControl(http.FileServer(http.Dir("ui"))))

// NEW: Serves from root dist/
mux.Handle("/", serveWithCacheControl(http.FileServer(http.Dir("dist"))))
```

### 2. Cache Busting Headers

**File:** `web/server.go` - `middlewareCORS`

Added aggressive cache-busting for ALL responses:

```go
func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Force cache busting for ALL responses
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		
		// CORS headers...
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

### 3. API Debug Logging

All Bambu Lab API endpoints now log with `[API DEBUG]` prefix:

**Examples:**
```
[API DEBUG] Bambu login request received
[API DEBUG] Bambu login: Saving account for user: test@example.com (region: US)
[API DEBUG] Bambu login: Account saved successfully
[API DEBUG] Bambu 2FA verification request received
[API DEBUG] Bambu 2FA: Verifying code for user: test@example.com
[API DEBUG] Bambu 2FA: Verification successful
[API DEBUG] Bambu discover printers request received
[API DEBUG] Bambu discover: No email provided, using default account
[API DEBUG] Bambu discover: Using default account: test@example.com
[API DEBUG] Bambu discover: Creating provider for user: test@example.com (region: US)
[API DEBUG] Bambu discover: Attempting authentication...
[API DEBUG] Bambu discover: Authentication successful, discovering printers...
[API DEBUG] Bambu discover: Found 3 printers
```

---

## 🎨 UX Improvements

### Bambu Lab Add Printer Modal

The modal now correctly shows:

1. **Connection Mode Toggle:**
   - 🏠 **LAN (Local)** - Direct printer connection
   - ☁️ **Cloud (Bambu)** - Cloud account integration

2. **LAN Mode Fields:**
   - Printer IP Address
   - Access Code (6-character from printer screen)

3. **Cloud Mode Flow:**
   - Email/Password login
   - Region selection (US/EU/CN)
   - 2FA verification modal (if required)
   - Printer discovery list
   - One-click binding to cluster

### 2FA Handling

The UI correctly catches 401 errors and triggers the 2FA input modal:

```javascript
async function handleCloudLogin() {
  try {
    await api.bambuLogin(cloudEmail, cloudPassword, cloudRegion);
    handleDiscover();
  } catch (err) {
    // Check if 2FA is required
    const result = await api.bambuDiscover(cloudEmail);
    if (result.tfa_required) {
      setShow2FA(true);  // Show 2FA modal
      return;
    }
    alert('Login failed: ' + err.message);
  }
}
```

---

## 📋 How to Run

### Make Script Executable (First Time Only)

```bash
chmod +x nexus-dev.sh
```

### Run the Script

```bash
./nexus-dev.sh
```

### Access the Application

- **Web UI:** http://localhost:8080
- **API:** http://localhost:8080/api
- **WebSocket:** ws://localhost:8080/ws

---

## 📁 Files Modified/Created

| File | Action | Description |
|------|--------|-------------|
| `nexus-dev.sh` | ✅ Created | Build & run automation script |
| `web/server.go` | ✅ Modified | Changed static path to `dist/`, added cache-busting, added debug logging |
| `QUICKSTART.md` | ✅ Created | Comprehensive quick start guide |
| `BUILD_SUMMARY.md` | ✅ Created | This file |

---

## 🎯 Testing Checklist

- [x] Script kills existing processes
- [x] Script removes old build artifacts
- [x] Frontend builds successfully
- [x] Assets copy to root `dist/` folder
- [x] `go mod tidy` runs successfully
- [x] Backend builds successfully
- [x] Application starts with debug logging
- [x] Web UI accessible at http://localhost:8080
- [x] Cache-busting headers present on all responses
- [x] API debug logging visible in console
- [x] Bambu Lab dual-mode UI shows correctly
- [x] 2FA modal triggers on authentication failure

---

## 🚀 Ready Product

The product is **ready to run** with a single command:

```bash
./nexus-dev.sh
```

This will:
1. Clean up any old processes/artifacts
2. Build the frontend
3. Deploy assets
4. Build the backend
5. Start the application

**No manual steps required!**
