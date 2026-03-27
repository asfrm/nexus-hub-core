#!/bin/bash

###############################################################################
# Nexus Hub Core - Development Build & Run Utility
# 
# This script automates the entire development lifecycle:
# 1. Cleanup old processes and build artifacts
# 2. Build frontend (React + Vite)
# 3. Deploy assets to root dist/ folder
# 4. Build Go backend
# 5. Run the application
#
# Usage: ./nexus-dev.sh
###############################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_ROOT"

echo -e "${BLUE}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Nexus Hub Core - Development Build & Run Utility    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

###############################################################################
# Step 1: Cleanup
###############################################################################
echo -e "${YELLOW}[1/6] Cleaning up old processes and artifacts...${NC}"

# Kill any existing nexus-hub-core processes
if pgrep -f "nexus-hub-core" > /dev/null; then
    echo "   Stopping existing nexus-hub-core processes..."
    pkill -f "nexus-hub-core" || true
    sleep 1
fi

# Remove old build artifacts
if [ -d "ui/dist" ]; then
    echo "   Removing ui/dist/..."
    rm -rf ui/dist
fi

if [ -d "dist" ]; then
    echo "   Removing root dist/..."
    rm -rf dist
fi

if [ -f "nexus-hub-core" ]; then
    echo "   Removing old binary..."
    rm -f nexus-hub-core
fi

echo -e "   ${GREEN}✓ Cleanup complete${NC}"
echo ""

###############################################################################
# Step 2: Frontend Build
###############################################################################
echo -e "${YELLOW}[2/6] Building frontend (React + Vite)...${NC}"

cd ui

# Check if node_modules exists, if not run npm install
if [ ! -d "node_modules" ]; then
    echo "   Installing npm dependencies..."
    npm install
fi

# Build frontend
echo "   Running npm run build..."
npm run build

if [ ! -d "dist" ]; then
    echo -e "   ${RED}✗ Frontend build failed - dist/ not created${NC}"
    exit 1
fi

echo -e "   ${GREEN}✓ Frontend build complete${NC}"
cd ..
echo ""

###############################################################################
# Step 3: Asset Deployment
###############################################################################
echo -e "${YELLOW}[3/6] Deploying assets to root dist/ folder...${NC}"

# Create root dist directory
mkdir -p dist

# Copy everything from ui/dist/ to root dist/
cp -r ui/dist/* dist/

if [ ! -f "dist/index.html" ]; then
    echo -e "   ${RED}✗ Asset deployment failed - index.html not found${NC}"
    exit 1
fi

echo -e "   ${GREEN}✓ Assets deployed to dist/${NC}"
echo ""

###############################################################################
# Step 4: Backend Prep
###############################################################################
echo -e "${YELLOW}[4/6] Preparing Go backend...${NC}"

# Run go mod tidy
echo "   Running go mod tidy..."
go mod tidy

echo -e "   ${GREEN}✓ Go modules updated${NC}"
echo ""

###############################################################################
# Step 5: Binary Compilation
###############################################################################
echo -e "${YELLOW}[5/6] Building Go backend...${NC}"

# Build the Go backend
echo "   Running go build -o nexus-hub-core..."
go build -o nexus-hub-core -v .

if [ ! -f "nexus-hub-core" ]; then
    echo -e "   ${RED}✗ Backend build failed${NC}"
    exit 1
fi

echo -e "   ${GREEN}✓ Backend build complete${NC}"
echo ""

###############################################################################
# Step 6: Execution
###############################################################################
echo -e "${YELLOW}[6/6] Starting Nexus Hub Core...${NC}"
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Build successful! Starting application...             ║${NC}"
echo -e "${GREEN}║                                                        ║${NC}"
echo -e "${GREEN}║  Web UI:    http://localhost:8080                      ║${NC}"
echo -e "${GREEN}║  API:       http://localhost:8080/api                  ║${NC}"
echo -e "${GREEN}║  WebSocket: ws://localhost:8080/ws                     ║${NC}"
echo -e "${GREEN}║                                                        ║${NC}"
echo -e "${GREEN}║  Press Ctrl+C to stop                                  ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════╝${NC}"
echo ""

# Start the application with debug flag
./nexus-hub-core -debug
