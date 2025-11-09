#!/bin/bash

# **********************************************************************
# -------------------------------------------------------------------
# Project Name : Abdal 4iProto Panel
# File Name    : uninstall-service.sh
# Author       : Ebrahim Shafiei (EbraSha)
# Email        : Prof.Shafiei@Gmail.com
# Created On   : 2025-11-09 18:28:25
# Description  : Uninstallation script for Abdal 4iProto Panel systemd service
# -------------------------------------------------------------------
#
# "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
# – Ebrahim Shafiei
#
# **********************************************************************

PANEL_SERVICE_NAME="abdal-4iproto-panel"
SERVER_SERVICE_NAME="abdal-4iproto-server"
INSTALL_DIR="/usr/local/abdal-4iproto-server"

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
    echo -e "  ${BOLD}sudo ./uninstall-service.sh${NC}"
    echo ""
    echo -e "${RED}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    exit 1
fi

# Function to uninstall service only
uninstall_service_only() {
    echo ""
    print_info "Uninstalling $PANEL_SERVICE_NAME service only..."
    echo ""
    
    # Stop service
    if systemctl is-active --quiet "$PANEL_SERVICE_NAME"; then
        systemctl stop "$PANEL_SERVICE_NAME"
        print_success "Stopped service: $PANEL_SERVICE_NAME"
    fi
    
    # Disable service
    if systemctl is-enabled --quiet "$PANEL_SERVICE_NAME"; then
        systemctl disable "$PANEL_SERVICE_NAME"
        print_success "Disabled service: $PANEL_SERVICE_NAME"
    fi
    
    # Remove service file
    if [ -f "/etc/systemd/system/$PANEL_SERVICE_NAME.service" ]; then
        rm "/etc/systemd/system/$PANEL_SERVICE_NAME.service"
        print_success "Removed service file: /etc/systemd/system/$PANEL_SERVICE_NAME.service"
    fi
    
    # Reload systemd
    systemctl daemon-reload
    print_info "Reloaded systemd daemon"
    
    echo ""
    print_success "Service uninstalled successfully!"
    echo ""
    echo -e "${BOLD}Programmer:${NC} ${GREEN}Ebrahim Shafiei (EbraSha)${NC}"
    echo -e "${BOLD}Email:${NC} ${CYAN}Prof.Shafiei@Gmail.com${NC}"
}

# Function to uninstall everything
uninstall_complete() {
    echo ""
    echo -e "${RED}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${RED}${BOLD}  WARNING: This will remove ALL Abdal 4iProto files and services!${NC}"
    echo -e "${RED}${BOLD}═══════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${YELLOW}This includes:${NC}"
    echo -e "  ${RED}•${NC} $PANEL_SERVICE_NAME service"
    echo -e "  ${RED}•${NC} $SERVER_SERVICE_NAME service"
    echo -e "  ${RED}•${NC} All files in ${BOLD}$INSTALL_DIR${NC}"
    echo ""
    read -p "$(echo -e ${YELLOW}Are you sure you want to continue? ${NC}$(echo -e ${BOLD}${RED}yes${NC})$(echo -e ${YELLOW}/no): ${NC})" -r
    echo
    
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        echo ""
        print_warning "Uninstallation cancelled."
        exit 0
    fi
    
    echo ""
    print_info "Starting complete uninstallation..."
    echo ""
    
    # Check if panel service is installed
    if systemctl list-unit-files 2>/dev/null | grep -q "^$PANEL_SERVICE_NAME.service"; then
        print_info "Found $PANEL_SERVICE_NAME service"
        
        # Stop panel service
        if systemctl is-active --quiet "$PANEL_SERVICE_NAME"; then
            systemctl stop "$PANEL_SERVICE_NAME"
            print_success "Stopped service: $PANEL_SERVICE_NAME"
        fi
        
        # Disable panel service
        if systemctl is-enabled --quiet "$PANEL_SERVICE_NAME"; then
            systemctl disable "$PANEL_SERVICE_NAME"
            print_success "Disabled service: $PANEL_SERVICE_NAME"
        fi
        
        # Remove panel service file
        if [ -f "/etc/systemd/system/$PANEL_SERVICE_NAME.service" ]; then
            rm "/etc/systemd/system/$PANEL_SERVICE_NAME.service"
            print_success "Removed service file: /etc/systemd/system/$PANEL_SERVICE_NAME.service"
        fi
    else
        print_warning "$PANEL_SERVICE_NAME service is not installed"
    fi
    
    # Check if server service is installed
    if systemctl list-unit-files 2>/dev/null | grep -q "^$SERVER_SERVICE_NAME.service"; then
        print_info "Found $SERVER_SERVICE_NAME service"
        
        # Stop server service
        if systemctl is-active --quiet "$SERVER_SERVICE_NAME"; then
            systemctl stop "$SERVER_SERVICE_NAME"
            print_success "Stopped service: $SERVER_SERVICE_NAME"
        fi
        
        # Disable server service
        if systemctl is-enabled --quiet "$SERVER_SERVICE_NAME"; then
            systemctl disable "$SERVER_SERVICE_NAME"
            print_success "Disabled service: $SERVER_SERVICE_NAME"
        fi
        
        # Remove server service file
        if [ -f "/etc/systemd/system/$SERVER_SERVICE_NAME.service" ]; then
            rm "/etc/systemd/system/$SERVER_SERVICE_NAME.service"
            print_success "Removed service file: /etc/systemd/system/$SERVER_SERVICE_NAME.service"
        fi
    else
        print_warning "$SERVER_SERVICE_NAME service is not installed"
    fi
    
    # Reload systemd
    systemctl daemon-reload
    print_info "Reloaded systemd daemon"
    
    # Remove installation directory
    if [ -d "$INSTALL_DIR" ]; then
        print_info "Removing installation directory: $INSTALL_DIR"
        rm -rf "$INSTALL_DIR"
        print_success "Removed installation directory: $INSTALL_DIR"
    else
        print_warning "Installation directory $INSTALL_DIR does not exist"
    fi
    
    echo ""
    print_success "Complete uninstallation finished successfully!"
    echo ""
    echo -e "${BOLD}Programmer:${NC} ${GREEN}Ebrahim Shafiei (EbraSha)${NC}"
    echo -e "${BOLD}Email:${NC} ${CYAN}Prof.Shafiei@Gmail.com${NC}"
}

# Main menu
echo ""
echo -e "${BLUE}${BOLD}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}${BOLD}        Abdal 4iProto Panel Uninstaller${NC}"
echo -e "${BLUE}${BOLD}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${WHITE}What do you want to uninstall?${NC}"
echo ""
echo -e "  ${GREEN}1)${NC} Service only (${CYAN}$PANEL_SERVICE_NAME${NC})"
echo -e "  ${GREEN}2)${NC} Complete removal (${RED}All services and files${NC})"
echo ""
read -p "$(echo -e ${YELLOW}Please select an option ${NC}$(echo -e ${BOLD}${GREEN}1${NC})$(echo -e ${YELLOW} or ${NC})$(echo -e ${BOLD}${GREEN}2${NC})$(echo -e ${YELLOW}): ${NC})" -r
echo

case $REPLY in
    1)
        uninstall_service_only
        ;;
    2)
        uninstall_complete
        ;;
    *)
        print_error "Invalid option. Exiting."
        exit 1
        ;;
esac
