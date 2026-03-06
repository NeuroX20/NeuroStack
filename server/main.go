package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nirodbx/neurostack/config"
	"github.com/nirodbx/neurostack/handler"
)

const banner = `
  _   _                      ____  _             _    
 | \ | | ___ _   _ _ __ ___ / ___|| |_ __ _  ___| | __
 |  \| |/ _ \ | | | '__/ _ \\___ \| __/ _` + "`" + ` |/ __| |/ /
 | |\  |  __/ |_| | | | (_) |___) | || (_| | (__|   < 
 |_| \_|\___|\__,_|_|  \___/|____/ \__\__,_|\___|_|\_\
`

func main() {
	cfg := config.Load()

	fmt.Println("\033[36m" + banner + "\033[0m")
	fmt.Println("\033[90m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Printf("  \033[1m%-20s\033[0m \033[33mv0.1.0\033[0m\n", "NeuroStack")
	fmt.Printf("  \033[90m%-20s\033[0m %s\n", "Local Dev Stack", "Go + MariaDB + phpMyAdmin")
	fmt.Println("\033[90m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Printf("  \033[90m%-20s\033[0m \033[32m%s\033[0m\n", "Dashboard", "http://"+cfg.Addr)
	fmt.Printf("  \033[90m%-20s\033[0m \033[32m%s\033[0m\n", "phpMyAdmin", "http://localhost:8888")
	fmt.Printf("  \033[90m%-20s\033[0m \033[32m%s:%s\033[0m\n", "MariaDB", cfg.DBHost, cfg.DBPort)
	fmt.Printf("  \033[90m%-20s\033[0m \033[32m%s\033[0m\n", "Started at", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println("\033[90m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("  \033[90mPress Ctrl+C to stop\033[0m")
	fmt.Println()

	mux := http.NewServeMux()

	// Dashboard
	mux.HandleFunc("/", handler.Dashboard)

	// API — Server
	mux.HandleFunc("/api/status", handler.Status)

	// API — Database
	mux.HandleFunc("/api/db/query", handler.DBQuery)
	mux.HandleFunc("/api/db/databases", handler.DBList)
	mux.HandleFunc("/api/db/tables", handler.DBTables)

	// API — File Manager
	mux.HandleFunc("/api/fm/list", handler.FMList)
	mux.HandleFunc("/api/fm/read", handler.FMRead)
	mux.HandleFunc("/api/fm/write", handler.FMWrite)
	mux.HandleFunc("/api/fm/delete", handler.FMDelete)
	mux.HandleFunc("/api/fm/mkdir", handler.FMMkdir)
	mux.HandleFunc("/api/fm/upload", handler.FMUpload)
	mux.HandleFunc("/api/fm/download", handler.FMDownload)
	mux.HandleFunc("/api/fm/zip", handler.FMZip)

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	log.Printf("[NeuroStack] Listening on http://%s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
		os.Exit(1)
	}
}
