#!/bin/bash

# NeuroStack Installer
# https://github.com/nirodbx/neurostack

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
MUTED='\033[0;90m'
BOLD='\033[1m'
NC='\033[0m'

NEURO_VERSION="0.1.0"
NEURO_DIR="$HOME/neurostack"
WWW_DIR="$NEURO_DIR/www"
DB_PASS="neurostack"

banner() {
  echo ""
  echo -e "${CYAN}"
  echo "  _   _                      ____  _             _    "
  echo " | \ | | ___ _   _ _ __ ___ / ___|| |_ __ _  ___| | __"
  echo " |  \| |/ _ \ | | | '__/ _ \\___ \| __/ _\` |/ __| |/ /"
  echo " | |\  |  __/ |_| | | | (_) |___) | || (_| | (__|   < "
  echo " |_| \_|\___|\__,_|_|  \___/|____/ \__\__,_|\___|_|\_\\"
  echo -e "${NC}"
  echo -e "${MUTED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "  ${BOLD}NeuroStack${NC} ${YELLOW}v${NEURO_VERSION}${NC}"
  echo -e "  ${MUTED}Local Development Server Stack${NC}"
  echo -e "${MUTED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
}

log() { echo -e "  ${GREEN}✓${NC} $1"; }
info() { echo -e "  ${CYAN}→${NC} $1"; }
warn() { echo -e "  ${YELLOW}!${NC} $1"; }
err() { echo -e "  ${RED}✗${NC} $1"; exit 1; }
step() { echo -e "\n  ${BOLD}$1${NC}"; }

check_os() {
  step "Checking system..."
  if [ -f /etc/os-release ]; then
    . /etc/os-release
    log "OS: $NAME"
  fi

  # Check if running in Termux
  if [ -n "$TERMUX_VERSION" ]; then
    log "Environment: Termux (Android)"
    TERMUX=true
  else
    TERMUX=false
  fi

  # Check arch
  ARCH=$(uname -m)
  log "Architecture: $ARCH"
}

install_deps() {
  step "Installing dependencies..."

  if [ "$TERMUX" = true ]; then
    info "Using pkg (Termux)..."
    pkg update -y -q
    pkg install -y mariadb nginx php-fpm curl wget golang 2>/dev/null || true
    log "Termux packages installed"
  else
    info "Using apt..."
    apt-get update -qq
    apt-get install -y -qq \
      mariadb-server \
      nginx \
      php \
      php-fpm \
      php-mysql \
      phpmyadmin \
      golang-go \
      curl \
      wget \
      --fix-missing 2>/dev/null || true
    log "System packages installed"
  fi
}

setup_mariadb() {
  step "Setting up MariaDB..."

  if [ "$TERMUX" = true ]; then
    mysql_install_db 2>/dev/null || true
    mysqld_safe &
    sleep 3
  else
    service mariadb start 2>/dev/null || mysqld_safe & sleep 3
  fi

  # Set root password
  mariadb -u root 2>/dev/null << SQLEOF || true
ALTER USER 'root'@'localhost' IDENTIFIED BY '${DB_PASS}';
CREATE DATABASE IF NOT EXISTS phpmyadmin;
CREATE USER IF NOT EXISTS 'phpmyadmin'@'localhost' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON phpmyadmin.*.TO 'phpmyadmin'@'localhost';
FLUSH PRIVILEGES;
SQLEOF

  log "MariaDB configured (password: ${DB_PASS})"
}

setup_nginx() {
  step "Setting up Nginx..."

  # phpMyAdmin config
  cat > /etc/nginx/sites-available/phpmyadmin << NGINXEOF
server {
    listen 8888;
    server_name localhost;
    root /usr/share/phpmyadmin;
    index index.php;
    location / { try_files \$uri \$uri/ =404; }
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php8.3-fpm.sock;
    }
}
NGINXEOF

  # www config
  cat > /etc/nginx/sites-available/www << NGINXEOF
server {
    listen 8080;
    server_name localhost;
    root ${WWW_DIR};
    index index.php index.html;
    location / { try_files \$uri \$uri/ =404; }
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php8.3-fpm.sock;
    }
}
NGINXEOF

  ln -sf /etc/nginx/sites-available/phpmyadmin /etc/nginx/sites-enabled/ 2>/dev/null || true
  ln -sf /etc/nginx/sites-available/www /etc/nginx/sites-enabled/ 2>/dev/null || true
  rm -f /etc/nginx/sites-enabled/default 2>/dev/null || true

  service nginx start 2>/dev/null || nginx 2>/dev/null || true
  service php8.3-fpm start 2>/dev/null || php-fpm8.3 -D 2>/dev/null || true

  log "Nginx configured"
}

setup_neurostack() {
  step "Setting up NeuroStack..."

  mkdir -p "$NEURO_DIR" "$WWW_DIR"

  # Create sample index.php
  cat > "$WWW_DIR/index.php" << 'PHPEOF'
<?php
echo "<h1>Welcome to NeuroStack!</h1>";
echo "<p>PHP version: " . phpversion() . "</p>";
echo "<p>Server time: " . date('Y-m-d H:i:s') . "</p>";
PHPEOF

  log "www directory created: $WWW_DIR"

  # Build NeuroStack server
  if [ -d "$NEURO_DIR/server" ]; then
    info "Building NeuroStack server..."
    cd "$NEURO_DIR/server"
    go build -o "$NEURO_DIR/neurostack" . 2>/dev/null && log "Server binary built" || warn "Build failed — run manually"
  fi

  # Create start script
  cat > "$NEURO_DIR/start.sh" << STARTEOF
#!/bin/bash
export NEURO_DB_PASS=${DB_PASS}
export NEURO_ADDR=localhost:7000

# Start services
service mariadb start 2>/dev/null || true
service nginx start 2>/dev/null || true
service php8.3-fpm start 2>/dev/null || true

# Start NeuroStack
cd $NEURO_DIR/server
NEURO_DB_PASS=${DB_PASS} go run main.go
STARTEOF
  chmod +x "$NEURO_DIR/start.sh"
  log "Start script created: $NEURO_DIR/start.sh"
}

print_summary() {
  echo ""
  echo -e "${MUTED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "  ${GREEN}${BOLD}Installation complete!${NC}"
  echo -e "${MUTED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
  echo -e "  ${MUTED}Dashboard${NC}     ${CYAN}http://localhost:7000${NC}"
  echo -e "  ${MUTED}phpMyAdmin${NC}    ${CYAN}http://localhost:8888${NC}"
  echo -e "  ${MUTED}Web Server${NC}    ${CYAN}http://localhost:8080${NC}"
  echo -e "  ${MUTED}MariaDB${NC}       ${CYAN}localhost:3306${NC}"
  echo ""
  echo -e "  ${MUTED}DB Password${NC}   ${YELLOW}${DB_PASS}${NC}"
  echo -e "  ${MUTED}www folder${NC}    ${YELLOW}${WWW_DIR}${NC}"
  echo ""
  echo -e "  ${BOLD}To start NeuroStack:${NC}"
  echo -e "  ${CYAN}bash ~/neurostack/start.sh${NC}"
  echo ""
  echo -e "${MUTED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
}

# Run installer
banner
check_os
install_deps
setup_mariadb
setup_nginx
setup_neurostack
print_summary
