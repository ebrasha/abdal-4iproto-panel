#!/bin/bash

# **********************************************************************
# -------------------------------------------------------------------
# Project Name : Abdal 4iProto Panel
# File Name    : install-service.sh
# Author       : Ebrahim Shafiei (EbraSha)
# Email        : Prof.Shafiei@Gmail.com
# Created On   : 2025-11-09 18:48:19
# Description  : Installation script for Abdal 4iProto Panel and Server as systemd services
# -------------------------------------------------------------------
#
# "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
# – Ebrahim Shafiei
#
# **********************************************************************

PANEL_SERVICE_NAME="abdal-4iproto-panel"
SERVER_SERVICE_NAME="abdal-4iproto-server"
PANEL_EXECUTABLE="abdal_4iproto_panel_linux"
SERVER_EXECUTABLE="abdal_4iproto_server_linux"
INSTALL_DIR="/usr/local/abdal-4iproto-server"
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Required files that must exist in current directory
REQUIRED_FILES=(
    "abdal-4iproto-panel.json"
    "abdal_4iproto_panel_linux"
    "abdal_4iproto_server_linux"
    "blocked_ips.json"
    "id_ed25519"
    "id_ed25519.pub"
    "server_config.json"
    "users.json"
)

# ANSI color codes for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Function to print colored messages
print_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root - MUST be root to continue
if [ "$EUID" -ne 0 ]; then 
    echo ""
    echo -e "${RED}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    print_error "This script must be run as root!"
    echo ""
    echo -e "${YELLOW}Please run with sudo:${NC}"
    echo -e "  ${BOLD}sudo ./install-service.sh${NC}"
    echo ""
    echo -e "${RED}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    exit 1
fi

# Check if services are already installed
echo ""
print_info "Checking if services are already installed..."
SERVICES_INSTALLED=false

if systemctl list-unit-files | grep -q "^${SERVER_SERVICE_NAME}.service" 2>/dev/null; then
    SERVICES_INSTALLED=true
fi

if systemctl list-unit-files | grep -q "^${PANEL_SERVICE_NAME}.service" 2>/dev/null; then
    SERVICES_INSTALLED=true
fi

if [ "$SERVICES_INSTALLED" = true ]; then
    echo ""
    echo -e "${YELLOW}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    print_warning "Services are already installed!"
    echo ""
    echo -e "${YELLOW}The following services are already installed:${NC}"
    if systemctl list-unit-files | grep -q "^${SERVER_SERVICE_NAME}.service" 2>/dev/null; then
        echo -e "  ${BOLD}• $SERVER_SERVICE_NAME${NC}"
    fi
    if systemctl list-unit-files | grep -q "^${PANEL_SERVICE_NAME}.service" 2>/dev/null; then
        echo -e "  ${BOLD}• $PANEL_SERVICE_NAME${NC}"
    fi
    echo ""
    echo -e "${YELLOW}If you want to reinstall, please uninstall first using:${NC}"
    echo -e "  ${BOLD}./uninstall-service.sh${NC}"
    echo ""
    echo -e "${YELLOW}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    exit 0
fi

print_success "Services are not installed. Proceeding with installation..."
echo ""

# Check if all required files exist
echo ""
echo -e "${BLUE}${BOLD}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}${BOLD}        Abdal 4iProto Panel & Server Installer${NC}"
echo -e "${BLUE}${BOLD}═══════════════════════════════════════════════════════════${NC}"
echo ""
print_info "Checking required files..."
echo ""

MISSING_FILES=()
for file in "${REQUIRED_FILES[@]}"; do
    if [ ! -f "$CURRENT_DIR/$file" ]; then
        MISSING_FILES+=("$file")
        print_error "Missing file: $file"
    else
        print_success "Found: $file"
    fi
done

if [ ${#MISSING_FILES[@]} -ne 0 ]; then
    echo ""
    print_error "Some required files are missing!"
    echo ""
    echo -e "${YELLOW}Please make sure all required files are in the current directory:${NC}"
    echo -e "  ${BOLD}$CURRENT_DIR${NC}"
    echo ""
    exit 1
fi

echo ""
print_success "All required files found!"
echo ""

# Create installation directory
print_info "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
print_success "Created directory: $INSTALL_DIR"
echo ""

# Copy all files to installation directory
print_info "Copying files to installation directory..."
for file in "${REQUIRED_FILES[@]}"; do
    cp "$CURRENT_DIR/$file" "$INSTALL_DIR/"
    print_success "Copied: $file"
done
echo ""

# Set executable permissions for executables
print_info "Setting executable permissions..."
chmod +x "$INSTALL_DIR/$PANEL_EXECUTABLE"
chmod +x "$INSTALL_DIR/$SERVER_EXECUTABLE"
print_success "Set executable permissions for $PANEL_EXECUTABLE"
print_success "Set executable permissions for $SERVER_EXECUTABLE"
echo ""

# Set proper permissions for all files
print_info "Setting file permissions..."
chmod 644 "$INSTALL_DIR/abdal-4iproto-panel.json"
chmod 644 "$INSTALL_DIR/blocked_ips.json"
chmod 644 "$INSTALL_DIR/server_config.json"
chmod 644 "$INSTALL_DIR/users.json"
chmod 600 "$INSTALL_DIR/id_ed25519"
chmod 644 "$INSTALL_DIR/id_ed25519.pub"
print_success "Set file permissions"
echo ""

# Create server service file
print_info "Creating server service file..."
cat > "/etc/systemd/system/$SERVER_SERVICE_NAME.service" << EOF
# -------------------------------------------------------------------
# Programmer       : Ebrahim Shafiei (EbraSha)
# Email            : Prof.Shafiei@Gmail.com
# -------------------------------------------------------------------
[Unit]
Description=Abdal 4iProto Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=-/etc/default/abdal-4iproto-server
ExecStart=$INSTALL_DIR/$SERVER_EXECUTABLE
Restart=always
RestartSec=3
LimitNOFILE=65536
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=$INSTALL_DIR
SyslogIdentifier=abdal-4iproto-server

[Install]
WantedBy=multi-user.target
EOF
print_success "Created server service file: /etc/systemd/system/$SERVER_SERVICE_NAME.service"
echo ""

# Create panel service file
print_info "Creating panel service file..."
cat > "/etc/systemd/system/$PANEL_SERVICE_NAME.service" << EOF
# -------------------------------------------------------------------
# Programmer       : Ebrahim Shafiei (EbraSha)
# Email            : Prof.Shafiei@Gmail.com
# -------------------------------------------------------------------
[Unit]
Description=Abdal 4iProto Panel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=-/etc/default/abdal-4iproto-panel
ExecStart=$INSTALL_DIR/$PANEL_EXECUTABLE
Restart=always
RestartSec=3
LimitNOFILE=65536
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=$INSTALL_DIR
SyslogIdentifier=abdal-4iproto-panel

[Install]
WantedBy=multi-user.target
EOF
print_success "Created panel service file: /etc/systemd/system/$PANEL_SERVICE_NAME.service"
echo ""

# Reload systemd
print_info "Reloading systemd daemon..."
systemctl daemon-reload
print_success "Reloaded systemd daemon"
echo ""

# Enable and start server service
print_info "Enabling server service: $SERVER_SERVICE_NAME"
systemctl enable "$SERVER_SERVICE_NAME"
print_success "Enabled server service: $SERVER_SERVICE_NAME"

print_info "Starting server service: $SERVER_SERVICE_NAME"
systemctl start "$SERVER_SERVICE_NAME"
print_success "Started server service: $SERVER_SERVICE_NAME"
echo ""

# Enable and start panel service
print_info "Enabling panel service: $PANEL_SERVICE_NAME"
systemctl enable "$PANEL_SERVICE_NAME"
print_success "Enabled panel service: $PANEL_SERVICE_NAME"

print_info "Starting panel service: $PANEL_SERVICE_NAME"
systemctl start "$PANEL_SERVICE_NAME"
print_success "Started panel service: $PANEL_SERVICE_NAME"
echo ""

# Installation complete message
echo ""
echo -e "${GREEN}${BOLD}═══════════════════════════════════════════════════════════${NC}"
print_success "Installation completed successfully!"
echo -e "${GREEN}${BOLD}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Service management commands
echo -e "${CYAN}${BOLD}Useful Commands:${NC}"
echo ""
echo -e "${WHITE}Server Service ($SERVER_SERVICE_NAME):${NC}"
echo -e "  ${GREEN}Check status:${NC}   systemctl status $SERVER_SERVICE_NAME"
echo -e "  ${GREEN}Start:${NC}          systemctl start $SERVER_SERVICE_NAME"
echo -e "  ${GREEN}Stop:${NC}           systemctl stop $SERVER_SERVICE_NAME"
echo -e "  ${GREEN}Restart:${NC}        systemctl restart $SERVER_SERVICE_NAME"
echo -e "  ${GREEN}View logs:${NC}      journalctl -u $SERVER_SERVICE_NAME -f"
echo ""
echo -e "${WHITE}Panel Service ($PANEL_SERVICE_NAME):${NC}"
echo -e "  ${GREEN}Check status:${NC}   systemctl status $PANEL_SERVICE_NAME"
echo -e "  ${GREEN}Start:${NC}          systemctl start $PANEL_SERVICE_NAME"
echo -e "  ${GREEN}Stop:${NC}           systemctl stop $PANEL_SERVICE_NAME"
echo -e "  ${GREEN}Restart:${NC}        systemctl restart $PANEL_SERVICE_NAME"
echo -e "  ${GREEN}View logs:${NC}      journalctl -u $PANEL_SERVICE_NAME -f"
echo ""
echo -e "${YELLOW}${BOLD}How to restart services:${NC}"
echo -e "  ${BOLD}Restart server:${NC}  systemctl restart $SERVER_SERVICE_NAME"
echo -e "  ${BOLD}Restart panel:${NC}  systemctl restart $PANEL_SERVICE_NAME"
echo -e "  ${BOLD}Restart both:${NC}   systemctl restart $SERVER_SERVICE_NAME $PANEL_SERVICE_NAME"
echo ""
echo -e "${YELLOW}Uninstall:${NC}        ./uninstall-service.sh"
echo ""

# Programmer information
echo -e "${BOLD}Programmer:${NC} ${GREEN}Ebrahim Shafiei (EbraSha)${NC}"
echo -e "${BOLD}Email:${NC} ${CYAN}Prof.Shafiei@Gmail.com${NC}"
echo ""
