#!/bin/bash
# OMA API Service Setup Script
# Following project rules: proper deployment, clean installation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Create oma user and group
create_user() {
    log_info "Creating oma user and group..."
    
    if ! getent group oma > /dev/null 2>&1; then
        groupadd --system oma
        log_success "Created oma group"
    else
        log_info "oma group already exists"
    fi
    
    if ! getent passwd oma > /dev/null 2>&1; then
        useradd --system --gid oma --home-dir /opt/migratekit --shell /bin/false oma
        log_success "Created oma user"
    else
        log_info "oma user already exists"
    fi
}

# Create directories
create_directories() {
    log_info "Creating directories..."
    
    mkdir -p /opt/migratekit/{bin,logs,config}
    mkdir -p /var/log/migratekit
    
    chown -R oma:oma /opt/migratekit
    chown -R oma:oma /var/log/migratekit
    
    log_success "Created directories"
}

# Build and install OMA API binary
install_binary() {
    log_info "Building and installing OMA API binary..."
    
    cd "$PROJECT_ROOT"
    
    # Build the binary
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go first."
        exit 1
    fi
    
    log_info "Building OMA API binary from consolidated source..."
    cd source/current/oma
    go build -ldflags="-s -w" -o oma-api ./cmd
    
    # Install binary
    cp oma-api /opt/migratekit/bin/
    chmod +x /opt/migratekit/bin/oma-api
    chown oma:oma /opt/migratekit/bin/oma-api
    
    log_success "Installed OMA API binary"
}

# Install systemd service
install_service() {
    log_info "Installing systemd service..."
    
    cp "$SCRIPT_DIR/oma-api.service" /etc/systemd/system/
    
    systemctl daemon-reload
    systemctl enable oma-api.service
    
    log_success "Installed and enabled oma-api service"
}

# Setup MariaDB database (optional)
setup_database() {
    read -p "Do you want to set up MariaDB database? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "Setting up MariaDB database..."
        
        # Install MariaDB if not present
        if ! command -v mysql &> /dev/null; then
            log_info "Installing MariaDB..."
            apt-get update
            apt-get install -y mariadb-server mariadb-client
            systemctl enable mariadb
            systemctl start mariadb
        fi
        
        # Create database and user
        log_info "Creating database and user..."
        mysql -u root <<EOF
CREATE DATABASE IF NOT EXISTS migratekit_oma CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE USER IF NOT EXISTS 'oma_user'@'localhost' IDENTIFIED BY 'oma_password';
GRANT ALL PRIVILEGES ON migratekit_oma.* TO 'oma_user'@'localhost';
FLUSH PRIVILEGES;
EOF
        
        log_success "MariaDB database setup complete"
    else
        log_info "Skipping database setup"
    fi
}

# Configure firewall (if UFW is installed)
configure_firewall() {
    if command -v ufw &> /dev/null; then
        log_info "Configuring firewall..."
        ufw allow 8080/tcp comment "OMA API"
        log_success "Firewall configured"
    fi
}

# Main installation function
main() {
    log_info "Starting OMA API service setup..."
    
    check_root
    create_user
    create_directories
    install_binary
    install_service
    setup_database
    configure_firewall
    
    log_success "OMA API service setup complete!"
    echo
    log_info "You can now:"
    log_info "  Start the service: systemctl start oma-api"
    log_info "  Check status:      systemctl status oma-api"
    log_info "  View logs:         journalctl -u oma-api -f"
    log_info "  Access API:        http://localhost:8080/health"
    log_info "  Swagger docs:      http://localhost:8080/swagger/"
    echo
    log_warning "Remember to configure your MariaDB password and other settings!"
}

# Run main function
main "$@"