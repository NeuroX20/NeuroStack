package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var startTime = time.Now()

func dbConn() (*sql.DB, error) {
	host := getEnv("NEURO_DB_HOST", "127.0.0.1")
	port := getEnv("NEURO_DB_PORT", "3306")
	user := getEnv("NEURO_DB_USER", "root")
	pass := getEnv("NEURO_DB_PASS", "")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/", user, pass, host, port)
	return sql.Open("mysql", dsn)
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Status(w http.ResponseWriter, r *http.Request) {
	db, err := dbConn()
	dbStatus := "connected"
	if err != nil {
		dbStatus = "error: " + err.Error()
	} else {
		if err := db.Ping(); err != nil {
			dbStatus = "disconnected: " + err.Error()
		}
		defer db.Close()
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"version":  "0.1.0",
		"uptime":   time.Since(startTime).String(),
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"database": dbStatus,
	})
}

func DBList(w http.ResponseWriter, r *http.Request) {
	db, err := dbConn()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer db.Close()
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	var databases []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		databases = append(databases, name)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"databases": databases})
}

func DBTables(w http.ResponseWriter, r *http.Request) {
	dbName := r.URL.Query().Get("db")
	if dbName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "db parameter required"})
		return
	}
	db, err := dbConn()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer db.Close()
	rows, err := db.Query(fmt.Sprintf("SHOW TABLES FROM `%s`", dbName))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"tables": tables})
}

func DBQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	var body struct {
		Query string `json:"query"`
		DB    string `json:"db"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	db, err := dbConn()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer db.Close()
	if body.DB != "" {
		if _, err := db.Exec(fmt.Sprintf("USE `%s`", body.DB)); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	rows, err := db.Query(body.Query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var results []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		row := make(map[string]interface{})
		for i, col := range cols {
			val := vals[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"columns": cols,
		"rows":    results,
		"count":   len(results),
	})
}

func Dashboard(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0">
  <title>NeuroStack</title>
  <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&family=Syne:wght@600;700;800&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #080810;
      --surface: #10101a;
      --surface2: #18182a;
      --border: #252538;
      --accent: #00e5ff;
      --accent2: #7c3aed;
      --green: #00ff88;
      --red: #ff4466;
      --yellow: #ffd700;
      --text: #e8e8f4;
      --muted: #5a5a7a;
    }
    * { margin:0; padding:0; box-sizing:border-box; -webkit-tap-highlight-color:transparent; }
    html, body { height:100%; overflow:hidden; }

    body {
      background: var(--bg);
      color: var(--text);
      font-family: 'Syne', sans-serif;
      display: flex;
      flex-direction: column;
    }

    /* Top nav bar (mobile-first) */
    .topbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 0 16px;
      height: 56px;
      background: var(--surface);
      border-bottom: 1px solid var(--border);
      flex-shrink: 0;
      position: relative;
      z-index: 100;
    }

    .logo-text {
      font-size: 20px;
      font-weight: 800;
      background: linear-gradient(135deg, var(--accent), var(--accent2));
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
    }

    .logo-badge {
      font-size: 10px;
      font-family: 'JetBrains Mono', monospace;
      color: var(--muted);
      margin-left: 8px;
      -webkit-text-fill-color: var(--muted);
    }

    .topbar-right {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .btn-pma {
      background: linear-gradient(135deg, #f97316, #ef4444);
      border: none;
      border-radius: 8px;
      color: white;
      font-size: 12px;
      font-weight: 700;
      font-family: 'Syne', sans-serif;
      padding: 7px 12px;
      cursor: pointer;
      text-decoration: none;
      display: flex;
      align-items: center;
      gap: 5px;
      white-space: nowrap;
    }

    /* Bottom tab bar */
    .tabbar {
      display: flex;
      background: var(--surface);
      border-top: 1px solid var(--border);
      flex-shrink: 0;
    }

    .tab {
      flex: 1;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 10px 4px;
      cursor: pointer;
      color: var(--muted);
      font-size: 10px;
      font-weight: 600;
      gap: 3px;
      border: none;
      background: none;
      transition: color 0.15s;
      font-family: 'Syne', sans-serif;
      letter-spacing: 0.3px;
    }

    .tab .icon { font-size: 18px; }
    .tab.active { color: var(--accent); }
    .tab.active .tab-dot {
      width: 4px; height: 4px;
      background: var(--accent);
      border-radius: 50%;
      margin-top: 2px;
    }

    /* Main content area */
    .content {
      flex: 1;
      overflow-y: auto;
      overflow-x: hidden;
      -webkit-overflow-scrolling: touch;
    }

    /* Panels */
    .panel { display: none; padding: 16px; }
    .panel.active { display: block; }

    /* Section title */
    .section-title {
      font-size: 22px;
      font-weight: 800;
      margin-bottom: 4px;
      letter-spacing: -0.3px;
    }

    .section-sub {
      font-size: 11px;
      color: var(--muted);
      font-family: 'JetBrains Mono', monospace;
      margin-bottom: 16px;
    }

    /* Status row */
    .status-row {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 10px;
      margin-bottom: 14px;
    }

    .stat-card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 14px 12px;
      position: relative;
      overflow: hidden;
    }

    .stat-card::after {
      content: '';
      position: absolute;
      top: 0; left: 0; right: 0;
      height: 2px;
      background: linear-gradient(90deg, var(--accent), var(--accent2));
    }

    .stat-label {
      font-size: 9px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 1px;
      font-family: 'JetBrains Mono', monospace;
      margin-bottom: 6px;
    }

    .stat-val {
      font-size: 13px;
      font-weight: 700;
      font-family: 'JetBrains Mono', monospace;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .green { color: var(--green); }
    .red { color: var(--red); }
    .cyan { color: var(--accent); }

    /* Card */
    .card {
      background: var(--surface);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 14px;
      margin-bottom: 12px;
    }

    .card-title {
      font-size: 10px;
      font-weight: 700;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 1.5px;
      font-family: 'JetBrains Mono', monospace;
      margin-bottom: 12px;
    }

    /* DB chips */
    .db-chips { display: flex; flex-wrap: wrap; gap: 8px; }

    .db-chip {
      background: var(--surface2);
      border: 1px solid var(--border);
      border-radius: 6px;
      padding: 6px 10px;
      font-size: 12px;
      font-family: 'JetBrains Mono', monospace;
      color: var(--accent);
      cursor: pointer;
    }

    /* Quick links */
    .quick-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 10px;
    }

    .quick-card {
      background: var(--surface2);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 14px;
      text-decoration: none;
      color: var(--text);
      display: flex;
      flex-direction: column;
      gap: 6px;
      cursor: pointer;
    }

    .quick-card:active { opacity: 0.7; }
    .quick-icon { font-size: 22px; }
    .quick-name { font-size: 13px; font-weight: 700; }
    .quick-desc { font-size: 11px; color: var(--muted); font-family: 'JetBrains Mono', monospace; }

    /* SQL Query */
    .input {
      background: var(--surface2);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 10px 12px;
      color: var(--text);
      font-family: 'JetBrains Mono', monospace;
      font-size: 13px;
      outline: none;
      width: 100%;
    }

    .input:focus { border-color: var(--accent); }

    textarea.input {
      resize: vertical;
      min-height: 100px;
      line-height: 1.6;
      margin-bottom: 10px;
    }

    .btn-run {
      width: 100%;
      padding: 12px;
      background: var(--accent);
      color: #000;
      border: none;
      border-radius: 8px;
      font-size: 14px;
      font-weight: 700;
      font-family: 'Syne', sans-serif;
      cursor: pointer;
      margin-bottom: 14px;
    }

    .btn-run:active { opacity: 0.8; }

    /* Table */
    .table-wrap { overflow-x: auto; -webkit-overflow-scrolling: touch; }

    .result-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
      font-family: 'JetBrains Mono', monospace;
      min-width: 300px;
    }

    .result-table th {
      text-align: left;
      padding: 8px 10px;
      border-bottom: 2px solid var(--border);
      color: var(--accent);
      font-size: 10px;
      text-transform: uppercase;
      letter-spacing: 1px;
      white-space: nowrap;
    }

    .result-table td {
      padding: 8px 10px;
      border-bottom: 1px solid var(--border);
      white-space: nowrap;
    }

    .result-count {
      font-size: 11px;
      color: var(--muted);
      font-family: 'JetBrains Mono', monospace;
      margin-top: 8px;
    }

    .error-box {
      background: rgba(255,68,102,0.1);
      border: 1px solid rgba(255,68,102,0.3);
      border-radius: 8px;
      padding: 12px;
      color: var(--red);
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
    }

    /* Services */
    .service-item {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 0;
      border-bottom: 1px solid var(--border);
    }

    .service-item:last-child { border-bottom: none; }
    .service-name { font-size: 14px; font-weight: 700; }
    .service-port { font-size: 12px; color: var(--muted); font-family: 'JetBrains Mono', monospace; margin-top: 2px; }
    .service-badge { font-size: 11px; font-family: 'JetBrains Mono', monospace; color: var(--green); }

    /* Status grid */
    .status-info-grid { display: flex; flex-direction: column; gap: 8px; }

    .status-info-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 10px 12px;
      background: var(--surface2);
      border-radius: 8px;
    }

    .info-key { font-size: 12px; color: var(--muted); font-family: 'JetBrains Mono', monospace; }
    .info-val { font-size: 12px; font-weight: 600; font-family: 'JetBrains Mono', monospace; color: var(--accent); }

    /* Clock */
    #clock { font-family: 'JetBrains Mono', monospace; font-size: 11px; color: var(--muted); }

    /* File Manager */
    .fm-crumb { color: var(--accent); cursor: pointer; font-family: 'JetBrains Mono', monospace; font-size: 12px; }
    .fm-item { display:flex; align-items:center; gap:8px; padding:9px 4px; border-bottom:1px solid var(--border); }
    .fm-item:last-child { border-bottom: none; }
    .fm-icon { font-size: 16px; flex-shrink: 0; }
    .fm-name { flex:1; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; font-family:'JetBrains Mono',monospace; font-size:12px; }
    .fm-link { color: var(--accent); cursor: pointer; }
    .fm-meta { font-size:10px; color:var(--muted); font-family:'JetBrains Mono',monospace; white-space:nowrap; flex-shrink:0; }
    .fm-actions { display:flex; gap:4px; flex-shrink:0; }
    .fm-btn { background:var(--surface2); border:1px solid var(--border); border-radius:6px; padding:4px 7px; cursor:pointer; font-size:13px; }
    .fm-btn-del { border-color: rgba(255,68,102,0.3); }
    .fm-btn:active { opacity: 0.7; }
  </style>
</head>
<body>

  <!-- Top bar -->
  <div class="topbar">
    <div style="display:flex;align-items:baseline;gap:6px">
      <span class="logo-text">⚡ NeuroStack</span>
      <span class="logo-badge">v0.1.0</span>
    </div>
    <div class="topbar-right">
      <span id="clock"></span>
      <a href="http://localhost:8888" target="_blank" class="btn-pma">🐬 phpMyAdmin</a>
    </div>
  </div>

  <!-- Content -->
  <div class="content">

    <!-- Dashboard -->
    <div id="panel-dashboard" class="panel active">
      <div class="section-title">Dashboard</div>
      <div class="section-sub">System overview</div>

      <div class="status-row">
        <div class="stat-card">
          <div class="stat-label">Server</div>
          <div class="stat-val green">● Online</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Database</div>
          <div class="stat-val" id="db-stat">...</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Uptime</div>
          <div class="stat-val cyan" id="uptime-stat">—</div>
        </div>
      </div>

      <div class="card">
        <div class="card-title">Databases</div>
        <div class="db-chips" id="db-list">
          <span style="color:var(--muted);font-size:12px;font-family:'JetBrains Mono',monospace">Loading...</span>
        </div>
      </div>

      <div class="card">
        <div class="card-title">Quick Access</div>
        <div class="quick-grid">
          <a href="http://localhost:8888" target="_blank" class="quick-card">
            <span class="quick-icon">🐬</span>
            <span class="quick-name">phpMyAdmin</span>
            <span class="quick-desc">:8888</span>
          </a>
          <div class="quick-card" onclick="switchTab('query')">
            <span class="quick-icon">⚡</span>
            <span class="quick-name">SQL Query</span>
            <span class="quick-desc">Run queries</span>
          </div>
          <div class="quick-card" onclick="switchTab('status')">
            <span class="quick-icon">📡</span>
            <span class="quick-name">Status</span>
            <span class="quick-desc">System info</span>
          </div>
          <div class="quick-card" onclick="switchTab('services')">
            <span class="quick-icon">🔧</span>
            <span class="quick-name">Services</span>
            <span class="quick-desc">All ports</span>
          </div>
        </div>
      </div>
    </div>

    <!-- SQL Query -->
    <div id="panel-query" class="panel">
      <div class="section-title">SQL Query</div>
      <div class="section-sub">Execute MariaDB queries</div>

      <div class="card">
        <input id="query-db" type="text" placeholder="Database name (optional)" class="input" style="margin-bottom:10px">
        <textarea id="query-input" class="input" placeholder="SELECT * FROM users LIMIT 10;"></textarea>
        <button class="btn-run" onclick="runQuery()">▶ Run Query</button>

        <div class="card-title">Result</div>
        <div id="query-result">
          <span style="color:var(--muted);font-size:12px;font-family:'JetBrains Mono',monospace">Run a query to see results...</span>
        </div>
      </div>
    </div>

    <!-- Status -->
    <div id="panel-status" class="panel">
      <div class="section-title">Server Status</div>
      <div class="section-sub">Real-time system info</div>
      <div class="card">
        <div class="status-info-grid" id="status-info">
          <span style="color:var(--muted);font-size:12px;font-family:'JetBrains Mono',monospace">Loading...</span>
        </div>
      </div>
    </div>

    <!-- File Manager -->
    <div id="panel-files" class="panel">
      <div class="section-title">File Manager</div>
      <div class="section-sub" id="fm-breadcrumb" style="margin-bottom:12px">www</div>
      <div class="card" style="padding:10px 14px;">
        <div style="display:flex;gap:8px;margin-bottom:12px;flex-wrap:wrap;">
          <button class="btn-run" style="width:auto;padding:8px 14px;font-size:12px;" onclick="fmMkdir()">📁 New Folder</button>
          <button class="btn-run" style="width:auto;padding:8px 14px;font-size:12px;background:var(--accent2);" onclick="fmUploadFile()">⬆️ Upload</button>
          <input type="file" id="fm-upload-input" multiple style="display:none" onchange="fmDoUpload(this)">
        </div>
        <div id="fm-list">
          <span style="color:var(--muted);font-size:12px;font-family:'JetBrains Mono',monospace">Loading...</span>
        </div>
      </div>
    </div>

    <!-- Services -->
    <div id="panel-services" class="panel">
      <div class="section-title">Services</div>
      <div class="section-sub">Running service endpoints</div>
      <div class="card">
        <div class="service-item">
          <div>
            <div class="service-name">⚡ Go Web Server</div>
            <div class="service-port">localhost:7000</div>
          </div>
          <span class="service-badge">● running</span>
        </div>
        <div class="service-item">
          <div>
            <div class="service-name">🐬 phpMyAdmin</div>
            <div class="service-port">localhost:8888 via Nginx</div>
          </div>
          <span class="service-badge">● running</span>
        </div>
        <div class="service-item">
          <div>
            <div class="service-name">🗄️ MariaDB</div>
            <div class="service-port">localhost:3306</div>
          </div>
          <span class="service-badge" id="db-service-badge">● running</span>
        </div>
      </div>
    </div>

  </div>

  <!-- Bottom tab bar -->
  <div class="tabbar">
    <button class="tab active" id="tab-dashboard" onclick="switchTab('dashboard')">
      <span class="icon">🏠</span>
      <span>Home</span>
      <div class="tab-dot"></div>
    </button>
    <button class="tab" id="tab-query" onclick="switchTab('query')">
      <span class="icon">⚡</span>
      <span>Query</span>
    </button>
    <button class="tab" id="tab-status" onclick="switchTab('status')">
      <span class="icon">📡</span>
      <span>Status</span>
    </button>
    <button class="tab" id="tab-files" onclick="switchTab('files');fmLoad('/')">
      <span class="icon">📂</span>
      <span>Files</span>
    </button>
    <button class="tab" id="tab-services" onclick="switchTab('services')">
      <span class="icon">🔧</span>
      <span>Services</span>
    </button>
  </div>

  <script>
    function switchTab(name) {
      document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
      document.querySelectorAll('.tab').forEach(t => {
        t.classList.remove('active');
        const dot = t.querySelector('.tab-dot');
        if (dot) dot.remove();
      });
      document.getElementById('panel-' + name).classList.add('active');
      const tab = document.getElementById('tab-' + name);
      tab.classList.add('active');
      if (!tab.querySelector('.tab-dot')) {
        const dot = document.createElement('div');
        dot.className = 'tab-dot';
        tab.appendChild(dot);
      }
    }

    // Clock
    function updateClock() {
      const now = new Date();
      document.getElementById('clock').textContent =
        now.toLocaleTimeString('en-US', {hour12: false});
    }
    setInterval(updateClock, 1000);
    updateClock();

    // Load status
    async function loadStatus() {
      const res = await fetch('/api/status');
      const data = await res.json();

      document.getElementById('uptime-stat').textContent = data.uptime.split('.')[0];

      const dbEl = document.getElementById('db-stat');
      if (data.database === 'connected') {
        dbEl.textContent = '● OK';
        dbEl.className = 'stat-val green';
      } else {
        dbEl.textContent = '● Error';
        dbEl.className = 'stat-val red';
        document.getElementById('db-service-badge').textContent = '● error';
        document.getElementById('db-service-badge').style.color = 'var(--red)';
      }

      const grid = document.getElementById('status-info');
      grid.innerHTML = Object.entries(data).map(([k, v]) =>
        '<div class="status-info-item">' +
        '<span class="info-key">' + k + '</span>' +
        '<span class="info-val">' + v + '</span>' +
        '</div>'
      ).join('');
    }

    // Load databases
    async function loadDatabases() {
      const res = await fetch('/api/db/databases');
      const data = await res.json();
      const el = document.getElementById('db-list');
      if (data.databases && data.databases.length > 0) {
        el.innerHTML = data.databases.map(db =>
          '<span class="db-chip" onclick="document.getElementById(\'query-db\').value=\'' + db + '\';switchTab(\'query\')">' + db + '</span>'
        ).join('');
      } else {
        el.innerHTML = '<span style="color:var(--muted);font-size:12px;font-family:\'JetBrains Mono\',monospace">No databases</span>';
      }
    }

    // Run query
    async function runQuery() {
      const query = document.getElementById('query-input').value.trim();
      const db = document.getElementById('query-db').value.trim();
      if (!query) return;

      const el = document.getElementById('query-result');
      el.innerHTML = '<span style="color:var(--muted);font-size:12px;font-family:\'JetBrains Mono\',monospace">Running...</span>';

      const res = await fetch('/api/db/query', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({query, db})
      });
      const data = await res.json();

      if (data.error) {
        el.innerHTML = '<div class="error-box">⚠ ' + data.error + '</div>';
        return;
      }
      if (!data.rows || data.rows.length === 0) {
        el.innerHTML = '<span style="color:var(--muted);font-size:12px;font-family:\'JetBrains Mono\',monospace">No results (0 rows)</span>';
        return;
      }

      let html = '<div class="table-wrap"><table class="result-table"><thead><tr>';
      data.columns.forEach(col => { html += '<th>' + col + '</th>'; });
      html += '</tr></thead><tbody>';
      data.rows.forEach(row => {
        html += '<tr>';
        data.columns.forEach(col => {
          html += '<td>' + (row[col] !== null && row[col] !== undefined ? row[col] : '<span style="color:var(--muted)">NULL</span>') + '</td>';
        });
        html += '</tr>';
      });
      html += '</tbody></table></div><div class="result-count">' + data.count + ' rows</div>';
      el.innerHTML = html;
    }

    // ─── File Manager ───────────────────────────────────────────
    let currentPath = '/';

    async function fmLoad(path) {
      if (path !== undefined) currentPath = path;
      const res = await fetch('/api/fm/list?path=' + encodeURIComponent(currentPath));
      const data = await res.json();
      if (data.error) {
        document.getElementById('fm-list').innerHTML = '<div class="error-box">⚠ ' + data.error + '</div>';
        return;
      }

      // Breadcrumb
      const parts = currentPath.split('/').filter(Boolean);
      let breadHTML = '<span class="fm-crumb" onclick="fmLoad(\'/\')">www</span>';
      let built = '';
      parts.forEach(p => {
        built += '/' + p;
        const snap = built;
        breadHTML += ' <span style="color:var(--muted)">›</span> <span class="fm-crumb" onclick="fmLoad(\'' + snap + '\')">' + p + '</span>';
      });
      document.getElementById('fm-breadcrumb').innerHTML = breadHTML;

      // File list
      const files = data.files || [];
      if (files.length === 0) {
        document.getElementById('fm-list').innerHTML = '<div style="color:var(--muted);font-size:12px;font-family:\'JetBrains Mono\',monospace;padding:12px 0">Empty directory</div>';
        return;
      }

      let html = '';
      if (currentPath !== '/') {
        html += '<div class="fm-item" onclick="fmGoUp()"><span class="fm-icon">📁</span><span class="fm-name">..</span><span class="fm-meta"></span></div>';
      }

      files.forEach(f => {
        const icon = f.is_dir ? '📁' : fileIcon(f.ext);
        const size = f.is_dir ? '—' : formatSize(f.size);
        const itemPath = (currentPath === '/' ? '' : currentPath) + '/' + f.name;
        html += '<div class="fm-item">';
        html += '<span class="fm-icon">' + icon + '</span>';
        if (f.is_dir) {
          html += '<span class="fm-name fm-link" onclick="fmLoad(\'' + itemPath + '\')">' + f.name + '</span>';
        } else {
          html += '<span class="fm-name">' + f.name + '</span>';
        }
        html += '<span class="fm-meta">' + size + ' · ' + f.mod_time + '</span>';
        html += '<div class="fm-actions">';
        if (!f.is_dir) {
          html += '<button class="fm-btn" onclick="fmEdit(\'' + itemPath + '\',\'' + f.name + '\')">✏️</button>';
          html += '<button class="fm-btn" onclick="fmDownload(\'' + itemPath + '\')">⬇️</button>';
        } else {
          html += '<button class="fm-btn" onclick="fmZip(\'' + itemPath + '\')">🗜️</button>';
        }
        html += '<button class="fm-btn fm-btn-del" onclick="fmDelete(\'' + itemPath + '\',\'' + f.name + '\')">🗑️</button>';
        html += '</div>';
        html += '</div>';
      });

      document.getElementById('fm-list').innerHTML = html;
    }

    function fmGoUp() {
      const parts = currentPath.split('/').filter(Boolean);
      parts.pop();
      fmLoad(parts.length === 0 ? '/' : '/' + parts.join('/'));
    }

    function fileIcon(ext) {
      const map = {
        '.php': '🐘', '.html': '🌐', '.htm': '🌐', '.css': '🎨',
        '.js': '📜', '.json': '📋', '.sql': '🗄️', '.txt': '📄',
        '.md': '📝', '.zip': '🗜️', '.png': '🖼️', '.jpg': '🖼️',
        '.jpeg': '🖼️', '.gif': '🖼️', '.svg': '🖼️', '.pdf': '📕',
      };
      return map[ext] || '📄';
    }

    function formatSize(bytes) {
      if (bytes < 1024) return bytes + 'B';
      if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + 'KB';
      return (bytes / (1024 * 1024)).toFixed(1) + 'MB';
    }

    async function fmEdit(path, name) {
      const res = await fetch('/api/fm/read?path=' + encodeURIComponent(path));
      const data = await res.json();
      if (data.error) { alert('Error: ' + data.error); return; }

      document.getElementById('fm-editor-title').textContent = '✏️ ' + name;
      document.getElementById('fm-editor-path').value = path;
      document.getElementById('fm-editor-content').value = data.content;
      document.getElementById('fm-editor').style.display = 'flex';
    }

    async function fmSave() {
      const path = document.getElementById('fm-editor-path').value;
      const content = document.getElementById('fm-editor-content').value;
      const res = await fetch('/api/fm/write', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({path, content})
      });
      const data = await res.json();
      if (data.error) { alert('Error: ' + data.error); return; }
      document.getElementById('fm-editor').style.display = 'none';
      fmLoad();
    }

    function fmDownload(path) {
      window.open('/api/fm/download?path=' + encodeURIComponent(path));
    }

    async function fmDelete(path, name) {
      if (!confirm('Delete ' + name + '?')) return;
      const res = await fetch('/api/fm/delete', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({path})
      });
      fmLoad();
    }

    async function fmMkdir() {
      const name = prompt('Folder name:');
      if (!name) return;
      const path = (currentPath === '/' ? '' : currentPath) + '/' + name;
      await fetch('/api/fm/mkdir', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({path})
      });
      fmLoad();
    }

    async function fmZip(path) {
      const res = await fetch('/api/fm/zip', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({path})
      });
      const data = await res.json();
      if (data.error) { alert('Error: ' + data.error); return; }
      alert('Zipped: ' + data.file);
      fmLoad();
    }

    function fmUploadFile() {
      document.getElementById('fm-upload-input').click();
    }

    async function fmDoUpload(input) {
      const formData = new FormData();
      formData.append('path', currentPath);
      for (const f of input.files) {
        formData.append('files', f);
      }
      const res = await fetch('/api/fm/upload', { method: 'POST', body: formData });
      const data = await res.json();
      fmLoad();
      input.value = '';
    }

    loadStatus();
    loadDatabases();
    setInterval(loadStatus, 15000);
  </script>

  <!-- File Editor Modal -->
  <div id="fm-editor" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,0.85);z-index:999;flex-direction:column;padding:16px;">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px;">
      <span id="fm-editor-title" style="font-size:15px;font-weight:700;color:var(--accent)"></span>
      <div style="display:flex;gap:8px;">
        <button onclick="fmSave()" style="background:var(--green);color:#000;border:none;border-radius:8px;padding:8px 16px;font-weight:700;font-family:'Syne',sans-serif;cursor:pointer;">💾 Save</button>
        <button onclick="document.getElementById('fm-editor').style.display='none'" style="background:var(--surface2);color:var(--text);border:1px solid var(--border);border-radius:8px;padding:8px 16px;font-weight:700;font-family:'Syne',sans-serif;cursor:pointer;">✕ Close</button>
      </div>
    </div>
    <input type="hidden" id="fm-editor-path">
    <textarea id="fm-editor-content" style="flex:1;background:var(--surface);border:1px solid var(--border);border-radius:10px;padding:14px;color:var(--text);font-family:'JetBrains Mono',monospace;font-size:13px;line-height:1.7;outline:none;resize:none;width:100%;height:calc(100% - 60px);"></textarea>
  </div>

</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}
