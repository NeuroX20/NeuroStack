<div align="center">

# ⚡ NeuroStack

**A modern local development server stack**

Built for developers who want XAMPP-like simplicity — but modern, lightweight, and Android-ready.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Android-blue?style=flat)](https://github.com/nirodbx/neurostack)

</div>

---

## ✨ What is NeuroStack?

NeuroStack bundles everything you need for local web development into one stack:

| Service | Port | Description |
|---------|------|-------------|
| ⚡ **Dashboard** | :7000 | Web UI to manage everything |
| 🐬 **phpMyAdmin** | :8888 | Database management |
| 🌐 **Web Server** | :8080 | Serve your PHP/HTML files |
| 🗄️ **MariaDB** | :3306 | MySQL-compatible database |

---

## 🚀 Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/NeuroX20/neurostack/main/install.sh | bash
```

Then start:

```bash
bash ~/neurostack/start.sh
```

Open **http://localhost:7000** in your browser. 🎉

---

## 📱 Android Support (via Termux)

NeuroStack runs on Android through [Termux](https://termux.dev):

```bash
# Install Termux from F-Droid (not Play Store)
# Then inside Termux:
curl -fsSL https://raw.githubusercontent.com/nirodbx/neurostack/main/install.sh | bash
```

Access from your Android browser at **http://localhost:7000**

---

## 🗂️ Features

### Dashboard
- Real-time server & database status
- Live clock and uptime
- Quick access to all services

### File Manager
- Browse, upload, download files
- Edit files directly in browser
- Create/delete folders
- ZIP & extract archives
- Auto-extract uploaded ZIP files

### SQL Query
- Run SQL queries from the browser
- View results in a clean table
- Select database from dashboard

### phpMyAdmin Integration
- One-click access from dashboard
- Full database management UI

---

## 📁 Project Structure

```
neurostack/
├── server/              # Go web server + API
│   ├── main.go          # Entry point + routes
│   ├── config/          # Configuration
│   └── handler/
│       ├── handler.go   # Dashboard + DB + SQL handlers
│       └── filemanager.go # File manager API
├── www/                 # Your web files (like htdocs)
├── sdk/
│   └── python/          # Python client SDK (coming soon)
├── install.sh           # One-command installer
└── start.sh             # Start all services
```

---

## 🛠️ Manual Setup

```bash
# 1. Install dependencies
apt-get install -y mariadb-server nginx php php-fpm php-mysql phpmyadmin golang-go

# 2. Start services
service mariadb start
service nginx start
service php8.3-fpm start

# 3. Run NeuroStack
cd ~/neurostack/server
NEURO_DB_PASS=neurostack go run main.go
```

---

## 🗺️ Roadmap

- [x] Go web server + dashboard
- [x] MariaDB integration
- [x] phpMyAdmin via Nginx
- [x] File Manager (upload, edit, delete, zip)
- [x] SQL Query from browser
- [x] One-command install script
- [x] Android/Termux support
- [ ] Python SDK
- [ ] Landing page
- [ ] Auto-start on boot
- [ ] HTTPS support
- [ ] Multi-user support

---

## 📚 Tech Stack

- **Backend** — Go (zero dependencies web server)
- **Database** — MariaDB
- **DB UI** — phpMyAdmin
- **Web Server** — Nginx + PHP-FPM
- **Frontend** — Vanilla HTML/CSS/JS

---

<div align="center">
Made with ❤️ · <a href="https://github.com/nirodbx/neurostack">github.com/nirodbx/neurostack</a>
</div>

