/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : main.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2025-11-05 09:38:30
 * Description  : Main server file for Abdal 4iProto management panel with embedded resources
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

//go:embed css/*.css
//go:embed js/*.js
//go:embed img/*.png
//go:embed img/*.ico
//go:embed templates
//go:embed translations
var embeddedFiles embed.FS

const (
	panelConfigFile  = "abdal-4iproto-panel.json"
	configFile       = "server_config.json"
	usersFile        = "users.json"
	blockedIPsFile   = "blocked_ips.json"
	usersLogDir      = "users_log"
	blockedAccessDir = "blocked_access"
	usersTrafficDir  = "users_traffic"
	sessionsDBPath   = "data/sessions/sessions.db"
	sessionsTempDBPath = "data/sessions/sessions-temp.db"
	sessionsBucket   = "sessions"
	sessionsUpdateInterval = 15 * time.Second
	defaultPort         = "8080"
	windowsServerService = "Abdal4iProtoServer"
	linuxServerService   = "abdal-4iproto-server"
	windowsPanelService  = "Abdal4iProtoPanel"
	linuxPanelService    = "abdal-4iproto-panel"
)

// ServerConfig represents the server configuration
type ServerConfig struct {
	Ports            []int  `json:"ports"`
	Shell            string `json:"shell"`
	MaxAuthAttempts  int    `json:"max_auth_attempts"`
	ServerVersion    string `json:"server_version"`
	PrivateKeyFile   string `json:"private_key_file"`
	PublicKeyFile    string `json:"public_key_file"`
}

// User represents a user in the system
type User struct {
	Username         string   `json:"username"`
	Password         string   `json:"password"`
	Role             string   `json:"role"`
	BlockedDomains   []string `json:"blocked_domains"`
	BlockedIPs       []string `json:"blocked_ips"`
	Log              string   `json:"log"`
	MaxSessions      int      `json:"max_sessions"`
	SessionTTLSeconds int     `json:"session_ttl_seconds"`
	MaxSpeedKbps     int      `json:"max_speed_kbps"`
	MaxTotalMB       int      `json:"max_total_mb"`
}

// BlockedIPs represents the blocked IPs configuration
type BlockedIPs struct {
	Blocked []string `json:"blocked"`
}

// TrafficData represents user traffic information
type TrafficData struct {
	Username          string `json:"username"`
	IP                string `json:"ip"`
	LastBytesSent     int64  `json:"last_bytes_sent"`
	LastBytesReceived int64  `json:"last_bytes_received"`
	LastBytesTotal    int64  `json:"last_bytes_total"`
	TotalBytesSent    int64  `json:"total_bytes_sent"`
	TotalBytesReceived int64 `json:"total_bytes_received"`
	TotalBytes        int64  `json:"total_bytes"`
	LastTimestamp     string `json:"last_timestamp"`
}

// SessionInfo represents session information
type SessionInfo struct {
	SessionID     string `json:"session_id"`
	Username      string `json:"username"`
	IP            string `json:"ip"`
	ClientVersion string `json:"client_version"`
	CreatedAt     int64  `json:"created_at"` // Unix timestamp in seconds
	LastSeen      int64  `json:"last_seen"`  // Unix timestamp in seconds
	Revoked       bool   `json:"revoked"`
}

// PanelConfig represents the panel configuration
type PanelConfig struct {
	Port            int      `json:"port"`
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	Logging         bool     `json:"logging"`
	BlockedIPs      []string `json:"blocked_ips"`
	MaxLoginAttempts int    `json:"max_login_attempts"`
	LoginAttemptWindow int  `json:"login_attempt_window"` // in seconds - time window for counting attempts
	BlockDuration      int    `json:"block_duration"`      // in seconds - duration IP remains blocked after exceeding max attempts
	Theme              string `json:"theme"`              // theme name: normal, ebrasha-dark
}

// LoginAttempt tracks login attempts
type LoginAttempt struct {
	IP        string
	Attempts  int
	LastAttempt time.Time
	Blocked   bool
	BlockedUntil time.Time
}

// TranslationData holds translation strings
type TranslationData map[string]map[string]string

var translations TranslationData
var panelConfig *PanelConfig
var panelLogger *Logger
var sessionsUpdateTicker *time.Ticker
var sessionsUpdateStop chan bool
var loginAttempts = make(map[string]*LoginAttempt)
var loginAttemptsMutex sync.RWMutex

const (
	panelVersion = "2.28"
	panelAuthor  = "Ebrahim Shafiei (EbraSha)"
	serviceName  = "Abdal4iProtoPanel"
)

var (
	httpServer *http.Server
	serverStop chan bool
)

func main() {
	// Check if running as Windows service
	if runtime.GOOS == "windows" {
		if isWindowsService() {
			runWindowsService()
			return
		}
	}

	// Run in standalone mode
	runServer()
}

// runServer starts the HTTP server (used by both service and standalone modes)
func runServer() {
	// Load panel configuration
	var err error
	panelConfig, err = loadPanelConfig()
	if err != nil {
		log.Printf("Warning: Could not load panel config, using defaults: %v", err)
		panelConfig = &PanelConfig{
			Port:              8080,
			Username:          "admin",
			Password:          "admin123",
			Logging:           true,
			BlockedIPs:        []string{},
			MaxLoginAttempts:  5,
			LoginAttemptWindow: 300, // 5 minutes in seconds
			BlockDuration:      3600, // 1 hour in seconds
			Theme:              "normal", // default theme
		}
		// Save default config
		savePanelConfig(panelConfig)
	}

	// Initialize logger
	panelLogger = NewLogger(panelConfig.Logging)

	// Log startup
	panelLogger.Info("=== Abdal 4iProto Panel Starting ===")
	panelLogger.Info(fmt.Sprintf("Panel Configuration Loaded - Port: %d, Username: %s, Logging: %v", 
		panelConfig.Port, panelConfig.Username, panelConfig.Logging))

	// Load translations
	loadTranslations()

	// Setup routes
	mux := http.NewServeMux()

	// Static files
	staticFS, _ := fs.Sub(embeddedFiles, "css")
	mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.FS(staticFS))))
	
	staticJS, _ := fs.Sub(embeddedFiles, "js")
	mux.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.FS(staticJS))))
	
	mux.HandleFunc("/img/logo.png", serveLogo)
	mux.HandleFunc("/logo.png", serveLogo) // Backward compatibility
	mux.HandleFunc("/favicon.ico", serveFavicon)
	mux.HandleFunc("/theme.css", serveThemeCSS)

	// Authentication routes
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/logout", logoutHandler)

	// Main pages (protected)
	mux.HandleFunc("/", authMiddleware(dashboardHandler))
	mux.HandleFunc("/users", authMiddleware(usersHandler))
	mux.HandleFunc("/server-config", authMiddleware(serverConfigHandler))
	mux.HandleFunc("/blocked-ips", authMiddleware(blockedIPsHandler))
	mux.HandleFunc("/logs", authMiddleware(logsHandler))
	mux.HandleFunc("/blocked-access", authMiddleware(blockedAccessHandler))
	mux.HandleFunc("/traffic", authMiddleware(trafficHandler))
	mux.HandleFunc("/sessions", authMiddleware(sessionsHandler))
	mux.HandleFunc("/about", authMiddleware(aboutHandler))
	mux.HandleFunc("/panel-config", authMiddleware(panelConfigHandler))

	// API endpoints (protected)
	mux.HandleFunc("/api/users", authMiddleware(apiUsersHandler))
	mux.HandleFunc("/api/panel-config", authMiddleware(apiPanelConfigHandler))
	mux.HandleFunc("/api/users/", authMiddleware(apiUserDetailHandler))
	mux.HandleFunc("/api/server-config", authMiddleware(apiServerConfigHandler))
	mux.HandleFunc("/api/blocked-ips", authMiddleware(apiBlockedIPsHandler))
	mux.HandleFunc("/api/logs/", authMiddleware(apiLogsHandler))
	mux.HandleFunc("/api/blocked-access/", authMiddleware(apiBlockedAccessHandler))
	mux.HandleFunc("/api/traffic/", authMiddleware(apiTrafficHandler))
	mux.HandleFunc("/api/sessions", authMiddleware(apiSessionsHandler))
	mux.HandleFunc("/api/sessions/", authMiddleware(apiSessionDetailHandler))
	mux.HandleFunc("/api/restart-server-service", authMiddleware(apiRestartServerServiceHandler))
	mux.HandleFunc("/api/restart-panel-service", authMiddleware(apiRestartPanelServiceHandler))
	mux.HandleFunc("/api/translations", authMiddleware(apiTranslationsHandler))

	// Start server
	portStr := fmt.Sprintf("%d", panelConfig.Port)
	if portStr == "0" {
		portStr = os.Getenv("PORT")
		if portStr == "" {
			portStr = defaultPort
		}
	}

	panelLogger.Info(fmt.Sprintf("Starting Abdal 4iProto Panel server on port %s", portStr))
	panelLogger.Info(fmt.Sprintf("Access the panel at: http://localhost:%s", portStr))
	panelLogger.Info(fmt.Sprintf("Default credentials - Username: %s, Password: %s", panelConfig.Username, panelConfig.Password))
	
	log.Printf("Starting Abdal 4iProto Panel server on port %s", portStr)
	log.Printf("Access the panel at: http://localhost:%s", portStr)
	log.Printf("Default credentials - Username: %s, Password: %s", panelConfig.Username, panelConfig.Password)
	
	// Initialize sessions database copying
	panelLogger.Info("Initializing sessions database copy mechanism...")
	if err := copySessionsDB(); err != nil {
		panelLogger.Warning(fmt.Sprintf("Failed to copy sessions database on startup: %v", err))
	}
	
	// Start periodic sessions database update
	sessionsUpdateTicker = time.NewTicker(sessionsUpdateInterval)
	sessionsUpdateStop = make(chan bool)
	go updateSessionsDBPeriodically()
	panelLogger.Info(fmt.Sprintf("Sessions database will be updated every %v", sessionsUpdateInterval))

	// Wrap mux with logging middleware
	loggedMux := loggingMiddleware(mux)
	
	// Create HTTP server
	httpServer = &http.Server{
		Addr:    ":" + portStr,
		Handler: loggedMux,
	}

	// Setup cleanup on shutdown
	serverStop = make(chan bool)
	defer cleanup()

	// Start server
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panelLogger.Error("Server failed to start", err)
		log.Fatalf("Server failed to start: %v", err)
	}
}

// cleanup performs cleanup operations on shutdown
func cleanup() {
	panelLogger.Info("Panel shutting down...")
	
	// Stop HTTP server gracefully
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			panelLogger.Warning(fmt.Sprintf("Error shutting down HTTP server: %v", err))
		}
	}
	
	// Stop sessions update ticker
	if sessionsUpdateTicker != nil {
		sessionsUpdateTicker.Stop()
	}
	
	// Close sessions update stop channel
	if sessionsUpdateStop != nil {
		close(sessionsUpdateStop)
	}
	
	// Remove temporary sessions database
	if err := os.Remove(sessionsTempDBPath); err != nil && !os.IsNotExist(err) {
		panelLogger.Warning(fmt.Sprintf("Failed to remove temp sessions database: %v", err))
	}
	
	// Close logger
	panelLogger.Close()
}


// loggingMiddleware logs all HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a ResponseWriter wrapper to capture status code
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Process request
		next.ServeHTTP(ww, r)
		
		// Log request
		duration := time.Since(start)
		panelLogger.Request(r.Method, r.URL.Path, r.RemoteAddr, ww.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// serveLogo serves the logo image
func serveLogo(w http.ResponseWriter, r *http.Request) {
	// Get theme from panel config
	theme := "normal"
	if panelConfig != nil && panelConfig.Theme != "" {
		theme = panelConfig.Theme
	}
	
	// Serve logo based on theme
	var logoFile string
	if theme == "ebrasha-dark" {
		logoFile = "img/ebrasha-dark-logo.png"
	} else {
		logoFile = "img/logo.png"
	}
	
	logoData, err := embeddedFiles.ReadFile(logoFile)
	if err != nil {
		// Fallback to default logo if theme logo not found
		logoData, err = embeddedFiles.ReadFile("img/logo.png")
		if err != nil {
			http.Error(w, "Logo not found", http.StatusNotFound)
			return
		}
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(logoData)
}

// serveFavicon serves the favicon based on current theme
func serveFavicon(w http.ResponseWriter, r *http.Request) {
	// Get theme from panel config
	theme := "normal"
	if panelConfig != nil && panelConfig.Theme != "" {
		theme = panelConfig.Theme
	}
	
	// Serve favicon based on theme
	var faviconFile string
	if theme == "ebrasha-dark" {
		faviconFile = "img/ebrasha-dark.ico"
	} else {
		faviconFile = "img/icon.ico"
	}
	
	faviconData, err := embeddedFiles.ReadFile(faviconFile)
	if err != nil {
		// Fallback to default favicon if theme favicon not found
		faviconData, err = embeddedFiles.ReadFile("img/icon.ico")
		if err != nil {
			http.Error(w, "Favicon not found", http.StatusNotFound)
			return
		}
	}
	w.Header().Set("Content-Type", "image/x-icon")
	w.Write(faviconData)
}

// serveThemeCSS serves the theme CSS file based on current theme
func serveThemeCSS(w http.ResponseWriter, r *http.Request) {
	// Get theme from panel config
	theme := "normal"
	if panelConfig != nil && panelConfig.Theme != "" {
		theme = panelConfig.Theme
	}
	
	// Serve theme CSS file
	cssFile := fmt.Sprintf("css/theme-%s.css", theme)
	cssData, err := embeddedFiles.ReadFile(cssFile)
	if err != nil {
		// Fallback to normal theme if theme CSS not found
		cssData, err = embeddedFiles.ReadFile("css/theme-normal.css")
		if err != nil {
			http.Error(w, "Theme CSS not found", http.StatusNotFound)
			return
		}
	}
	w.Header().Set("Content-Type", "text/css")
	w.Write(cssData)
}

// loadTranslations loads translation files
func loadTranslations() {
	translations = make(TranslationData)
	
	// Load English translations
	enData, err := embeddedFiles.ReadFile("translations/en.json")
	if err == nil {
		var enMap map[string]string
		if json.Unmarshal(enData, &enMap) == nil {
			translations["en"] = enMap
		}
	}

	// Load Persian/Farsi translations
	faData, err := embeddedFiles.ReadFile("translations/fa.json")
	if err == nil {
		var faMap map[string]string
		if json.Unmarshal(faData, &faMap) == nil {
			translations["fa"] = faMap
		}
	}
}

// getTranslation returns translated string for the given key
func getTranslation(lang, key string) string {
	if lang == "" {
		lang = "en"
	}
	if trans, ok := translations[lang]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	// Fallback to English
	if trans, ok := translations["en"]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	return key
}

// getLanguageFromRequest extracts language from request (cookie or query param)
func getLanguageFromRequest(r *http.Request) string {
	if lang := r.URL.Query().Get("lang"); lang != "" {
		return lang
	}
	if cookie, err := r.Cookie("lang"); err == nil {
		return cookie.Value
	}
	return "en"
}

// renderTemplate renders an HTML template with translations
func renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data map[string]interface{}) {
	lang := getLanguageFromRequest(r)
	
	tmplData := make(map[string]interface{})
	if data != nil {
		for k, v := range data {
			tmplData[k] = v
		}
	}
	tmplData["Lang"] = lang
	
	// Add current page path for sidebar active link
	currentPage := r.URL.Path
	if currentPage == "" {
		currentPage = "/"
	}
	tmplData["CurrentPage"] = currentPage
	
	// Create translation function
	tmplData["T"] = func(key string) string {
		return getTranslation(lang, key)
	}

	// Helper function to check if language is Persian
	tmplData["eq"] = func(a, b string) bool {
		return a == b
	}
	
	// Add panel version and author to all templates
	tmplData["PanelVersion"] = panelVersion
	tmplData["PanelAuthor"] = panelAuthor
	
	// Add theme to template data
	theme := "normal"
	if panelConfig != nil && panelConfig.Theme != "" {
		theme = panelConfig.Theme
	}
	tmplData["Theme"] = theme

	// Parse template with custom functions
	funcMap := template.FuncMap{
		"T": func(key string) string {
			return getTranslation(lang, key)
		},
		"eq": func(a, b string) bool {
			return a == b
		},
	}

	// Parse all templates and partials for includes
	tmpl := template.New(templateName).Funcs(funcMap)
	
	// Read and parse main template first
	mainTemplatePath := "templates/" + templateName
	mainTemplateContent, err := embeddedFiles.ReadFile(mainTemplatePath)
	if err != nil {
		panelLogger.Error(fmt.Sprintf("Template not found: %s", mainTemplatePath), err)
		http.Error(w, fmt.Sprintf("Template not found: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Parse main template
	if _, err := tmpl.Parse(string(mainTemplateContent)); err != nil {
		panelLogger.Error(fmt.Sprintf("Template parse error: %s", mainTemplatePath), err)
		http.Error(w, fmt.Sprintf("Template parse error: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Read and parse sidebar partial
	sidebarPath := "templates/partials/sidebar.html"
	sidebarContent, err := embeddedFiles.ReadFile(sidebarPath)
	if err != nil {
		// Skip sidebar if not found (for backward compatibility)
		panelLogger.Warning(fmt.Sprintf("Sidebar partial not found: %s", sidebarPath))
	} else {
		// Parse sidebar template with name "sidebar.html"
		if _, err := tmpl.New("sidebar.html").Parse(string(sidebarContent)); err != nil {
			panelLogger.Error(fmt.Sprintf("Sidebar template parse error: %s", sidebarPath), err)
			http.Error(w, fmt.Sprintf("Sidebar template parse error: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, tmplData); err != nil {
		panelLogger.Error(fmt.Sprintf("Template execution error: %s", templateName), err)
		http.Error(w, fmt.Sprintf("Template execution error: %v", err), http.StatusInternalServerError)
		return
	}
}

// Handlers for main pages
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "dashboard.html", nil)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "users.html", nil)
}

func serverConfigHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "server-config.html", nil)
}

func blockedIPsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "blocked-ips.html", nil)
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "logs.html", nil)
}

func blockedAccessHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "blocked-access.html", nil)
}

func trafficHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "traffic.html", nil)
}

func sessionsHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "sessions.html", nil)
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "about.html", nil)
}

func panelConfigHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "panel-config.html", nil)
}

// API handlers
func apiUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		users, err := loadUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		// Load traffic data for each user to show usage status
		usersWithTraffic := make([]map[string]interface{}, len(users))
		for i, user := range users {
			userMap := map[string]interface{}{
				"username":           user.Username,
				"password":           user.Password,
				"role":               user.Role,
				"blocked_domains":    user.BlockedDomains,
				"blocked_ips":        user.BlockedIPs,
				"log":                user.Log,
				"max_sessions":       user.MaxSessions,
				"session_ttl_seconds": user.SessionTTLSeconds,
				"max_speed_kbps":     user.MaxSpeedKbps,
				"max_total_mb":       user.MaxTotalMB,
			}
			
			// Load traffic data
			traffic, err := loadTrafficData(user.Username)
			if err == nil && traffic != nil {
				totalMB := float64(traffic.TotalBytes) / (1024 * 1024)
				maxMB := float64(user.MaxTotalMB)
				userMap["traffic_used_mb"] = totalMB
				userMap["traffic_limit_mb"] = maxMB
				userMap["traffic_exceeded"] = maxMB > 0 && totalMB > maxMB
			}
			
			usersWithTraffic[i] = userMap
		}
		
		json.NewEncoder(w).Encode(usersWithTraffic)

	case http.MethodPost:
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		users, err := loadUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if user exists
		for _, u := range users {
			if u.Username == user.Username {
				http.Error(w, "User already exists", http.StatusConflict)
				return
			}
		}

		users = append(users, user)
		if err := saveUsers(users); err != nil {
			panelLogger.Error("Failed to save users (POST)", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		panelLogger.Info(fmt.Sprintf("New user created: %s", user.Username))
		restartService()
		panelLogger.Info("Service restart triggered after user creation")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case http.MethodPut:
		// PUT to /api/users/username should be handled by apiUserDetailHandler
		http.Error(w, "Use PUT /api/users/username to update a user", http.StatusBadRequest)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiUserDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		users, err := loadUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, user := range users {
			if user.Username == username {
				json.NewEncoder(w).Encode(user)
				return
			}
		}
		http.Error(w, "User not found", http.StatusNotFound)

	case http.MethodPut:
		var user User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Ensure username matches the URL parameter
		if user.Username != username {
			http.Error(w, "Username mismatch", http.StatusBadRequest)
			return
		}

		users, err := loadUsers()
		if err != nil {
			panelLogger.Error("Failed to load users (PUT)", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		found := false
		for i, u := range users {
			if u.Username == username {
				users[i] = user
				found = true
				break
			}
		}

		if !found {
			panelLogger.Warning(fmt.Sprintf("User not found for update: %s", username))
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if err := saveUsers(users); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to save users (PUT): %s", username), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		panelLogger.Info(fmt.Sprintf("User updated: %s", username))
		restartService()
		panelLogger.Info("Service restart triggered after user update")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	case http.MethodDelete:
		users, err := loadUsers()
		if err != nil {
			panelLogger.Error("Failed to load users (DELETE)", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		found := false
		for i, user := range users {
			if user.Username == username {
				users = append(users[:i], users[i+1:]...)
				found = true
				break
			}
		}

		if !found {
			panelLogger.Warning(fmt.Sprintf("User not found for deletion: %s", username))
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if err := saveUsers(users); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to save users (DELETE): %s", username), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		panelLogger.Info(fmt.Sprintf("User deleted: %s", username))
		restartService()
		panelLogger.Info("Service restart triggered after user deletion")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiServerConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		config, err := loadServerConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(config)

	case http.MethodPut:
		var config ServerConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := saveServerConfig(&config); err != nil {
			panelLogger.Error("Failed to save server config", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		panelLogger.Info("Server configuration updated")
		restartService()
		panelLogger.Info("Service restart triggered after server config update")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// apiPanelConfigHandler handles panel configuration API requests
func apiPanelConfigHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		// Reload config from file to get latest data
		config, err := loadPanelConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(config)

	case http.MethodPut:
		var config PanelConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := savePanelConfig(&config); err != nil {
			panelLogger.Error("Failed to save panel config", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update global config
		panelConfig = &config
		
		// Update logger based on new logging setting
		panelLogger = NewLogger(config.Logging)

		panelLogger.Info("Panel configuration updated")
		
		// Send success response first
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		
		// Restart panel service in a goroutine (after response is sent)
		go func() {
			time.Sleep(500 * time.Millisecond) // Small delay to ensure response is sent
			restartPanelService()
			panelLogger.Info("Panel service restart triggered after config update")
		}()

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiBlockedIPsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		blockedIPs, err := loadBlockedIPs()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(blockedIPs)

	case http.MethodPut:
		var blockedIPs BlockedIPs
		if err := json.NewDecoder(r.Body).Decode(&blockedIPs); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := saveBlockedIPs(&blockedIPs); err != nil {
			panelLogger.Error("Failed to save blocked IPs", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		panelLogger.Info(fmt.Sprintf("Blocked IPs updated: %d IPs", len(blockedIPs.Blocked)))
		restartService()
		panelLogger.Info("Service restart triggered after blocked IPs update")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiLogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := strings.TrimPrefix(r.URL.Path, "/api/logs/")
	if username == "" {
		// Return list of available log files
		logFiles, err := getLogFiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(logFiles)
		return
	}

	// Read log file
	logPath := filepath.Join(usersLogDir, username+".log")
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, "Log file not found", http.StatusNotFound)
		return
	}

	lines := strings.Split(string(logContent), "\n")
	var logEntries []map[string]string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse log entry: [📡 User Access] [2025-11-04 15:26:49] Target: ogs.google.com | User IP: 5.119.165.209
		parts := strings.Split(line, "]")
		if len(parts) >= 3 {
			timestamp := strings.TrimSpace(strings.TrimPrefix(parts[1], "["))
			rest := strings.Join(parts[2:], "]")
			targetParts := strings.Split(rest, "|")
			target := ""
			userIP := ""
			if len(targetParts) >= 1 {
				target = strings.TrimSpace(strings.TrimPrefix(targetParts[0], "Target:"))
			}
			if len(targetParts) >= 2 {
				userIP = strings.TrimSpace(strings.TrimPrefix(targetParts[1], "User IP:"))
			}
			logEntries = append(logEntries, map[string]string{
				"timestamp": timestamp,
				"target":    target,
				"user_ip":   userIP,
				"raw":       line,
			})
		} else {
			logEntries = append(logEntries, map[string]string{
				"raw": line,
			})
		}
	}

	json.NewEncoder(w).Encode(logEntries)
}

func apiBlockedAccessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := strings.TrimPrefix(r.URL.Path, "/api/blocked-access/")
	if username == "" {
		// Return list of available blocked access log files
		logFiles, err := getBlockedAccessLogFiles()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(logFiles)
		return
	}

	// Read blocked access log file
	logPath := filepath.Join(blockedAccessDir, username+".log")
	logContent, err := os.ReadFile(logPath)
	if err != nil {
		http.Error(w, "Log file not found", http.StatusNotFound)
		return
	}

	lines := strings.Split(string(logContent), "\n")
	var logEntries []map[string]string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Parse log entry: [🚫 Blocked Access] [2025-11-04 15:26:49] Target: ogs.google.com | User IP: 5.119.165.209
		parts := strings.Split(line, "]")
		if len(parts) >= 3 {
			timestamp := strings.TrimSpace(strings.TrimPrefix(parts[1], "["))
			rest := strings.Join(parts[2:], "]")
			targetParts := strings.Split(rest, "|")
			target := ""
			userIP := ""
			if len(targetParts) >= 1 {
				target = strings.TrimSpace(strings.TrimPrefix(targetParts[0], "Target:"))
			}
			if len(targetParts) >= 2 {
				userIP = strings.TrimSpace(strings.TrimPrefix(targetParts[1], "User IP:"))
			}
			logEntries = append(logEntries, map[string]string{
				"timestamp": timestamp,
				"target":    target,
				"user_ip":   userIP,
				"raw":       line,
			})
		} else {
			logEntries = append(logEntries, map[string]string{
				"raw": line,
			})
		}
	}

	json.NewEncoder(w).Encode(logEntries)
}

func apiTrafficHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	username := strings.TrimPrefix(r.URL.Path, "/api/traffic/")
	if username == "" {
		// Return list of all users with traffic data
		users, err := loadUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var trafficList []map[string]interface{}
		for _, user := range users {
			traffic, err := loadTrafficData(user.Username)
			if err == nil && traffic != nil {
				trafficMap := map[string]interface{}{
					"username":            traffic.Username,
					"ip":                  traffic.IP,
					"last_bytes_sent":     traffic.LastBytesSent,
					"last_bytes_received": traffic.LastBytesReceived,
					"last_bytes_total":    traffic.LastBytesTotal,
					"total_bytes_sent":    traffic.TotalBytesSent,
					"total_bytes_received": traffic.TotalBytesReceived,
					"total_bytes":         traffic.TotalBytes,
					"last_timestamp":      traffic.LastTimestamp,
					"total_mb":            float64(traffic.TotalBytes) / (1024 * 1024),
					"max_total_mb":        user.MaxTotalMB,
					"exceeded":            user.MaxTotalMB > 0 && float64(traffic.TotalBytes)/(1024*1024) > float64(user.MaxTotalMB),
				}
				trafficList = append(trafficList, trafficMap)
			}
		}
		json.NewEncoder(w).Encode(trafficList)
		return
	}

	// Return specific user traffic
	traffic, err := loadTrafficData(username)
	if err != nil {
		http.Error(w, "Traffic data not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(traffic)
}

func apiTranslationsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}
	if trans, ok := translations[lang]; ok {
		json.NewEncoder(w).Encode(trans)
	} else {
		json.NewEncoder(w).Encode(translations["en"])
	}
}

func apiSessionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		sessions, err := loadAllSessions()
		if err != nil {
			panelLogger.Error("Failed to load sessions", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(sessions)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func apiSessionDetailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if err := deleteSession(sessionID); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to delete session: %s", sessionID), err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		panelLogger.Info(fmt.Sprintf("Session deleted: %s", sessionID))
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// File operations
func loadUsers() ([]User, error) {
	data, err := os.ReadFile(usersFile)
	if err != nil {
		return []User{}, nil
	}
	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func saveUsers(users []User) error {
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(usersFile, data, 0644)
}

func loadServerConfig() (*ServerConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return &ServerConfig{
			Ports:           []int{64235, 64236, 64237},
			Shell:           getDefaultShell(),
			MaxAuthAttempts: 3,
			ServerVersion:   "SSH-2.0-Abdal-4iProto-Server",
			PrivateKeyFile:  "id_rsa",
			PublicKeyFile:   "id_rsa.pub",
		}, nil
	}
	var config ServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	// Set default values if not present in config file
	if config.PrivateKeyFile == "" {
		config.PrivateKeyFile = "id_rsa"
	}
	if config.PublicKeyFile == "" {
		config.PublicKeyFile = "id_rsa.pub"
	}
	return &config, nil
}

func saveServerConfig(config *ServerConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func loadBlockedIPs() (*BlockedIPs, error) {
	data, err := os.ReadFile(blockedIPsFile)
	if err != nil {
		return &BlockedIPs{Blocked: []string{}}, nil
	}
	var blockedIPs BlockedIPs
	if err := json.Unmarshal(data, &blockedIPs); err != nil {
		return nil, err
	}
	return &blockedIPs, nil
}

func saveBlockedIPs(blockedIPs *BlockedIPs) error {
	data, err := json.MarshalIndent(blockedIPs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(blockedIPsFile, data, 0644)
}

func loadTrafficData(username string) (*TrafficData, error) {
	trafficPath := filepath.Join(usersTrafficDir, "traffic_"+username+".json")
	data, err := os.ReadFile(trafficPath)
	if err != nil {
		return nil, err
	}
	var traffic TrafficData
	if err := json.Unmarshal(data, &traffic); err != nil {
		return nil, err
	}
	return &traffic, nil
}

func getLogFiles() ([]string, error) {
	files, err := os.ReadDir(usersLogDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			username := strings.TrimSuffix(file.Name(), ".log")
			logFiles = append(logFiles, username)
		}
	}
	return logFiles, nil
}

// getBlockedAccessLogFiles returns a list of usernames that have blocked access log files
func getBlockedAccessLogFiles() ([]string, error) {
	files, err := os.ReadDir(blockedAccessDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".log") {
			username := strings.TrimSuffix(file.Name(), ".log")
			logFiles = append(logFiles, username)
		}
	}
	return logFiles, nil
}

// copySessionsDB copies the sessions database to a temporary file
func copySessionsDB() error {
	// Ensure directory exists
	dbDir := filepath.Dir(sessionsDBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Check if source database exists
	if _, err := os.Stat(sessionsDBPath); err != nil {
		if os.IsNotExist(err) {
			// Database doesn't exist, remove temp file if exists
			os.Remove(sessionsTempDBPath)
			return nil
		}
		return fmt.Errorf("failed to stat sessions database: %w", err)
	}

	// Remove old temp file if exists
	if err := os.Remove(sessionsTempDBPath); err != nil && !os.IsNotExist(err) {
		panelLogger.Warning(fmt.Sprintf("Failed to remove old temp sessions database: %v", err))
	}

	// Copy the database file
	sourceFile, err := os.Open(sessionsDBPath)
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(sessionsTempDBPath)
	if err != nil {
		return fmt.Errorf("failed to create temp database: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy database: %w", err)
	}

	// Set permissions on temp file
	if err := os.Chmod(sessionsTempDBPath, 0444); err != nil {
		panelLogger.Warning(fmt.Sprintf("Failed to set permissions on temp database: %v", err))
	}

	panelLogger.Debug("Sessions database copied successfully")
	return nil
}

// updateSessionsDBPeriodically updates the sessions database copy periodically
func updateSessionsDBPeriodically() {
	for {
		select {
		case <-sessionsUpdateTicker.C:
			if err := copySessionsDB(); err != nil {
				panelLogger.Warning(fmt.Sprintf("Failed to update sessions database copy: %v", err))
			}
		case <-sessionsUpdateStop:
			return
		}
	}
}

// loadAllSessions loads all sessions from the temporary bbolt database copy
func loadAllSessions() ([]SessionInfo, error) {
	var sessions []SessionInfo

	// Ensure directory exists
	dbDir := filepath.Dir(sessionsDBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		panelLogger.Error(fmt.Sprintf("Failed to create sessions directory: %s", dbDir), err)
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Check if temp database file exists and is readable
	if _, err := os.Stat(sessionsTempDBPath); err != nil {
		if os.IsNotExist(err) {
			// Temp database doesn't exist, try to copy it
			if copyErr := copySessionsDB(); copyErr != nil {
				panelLogger.Debug(fmt.Sprintf("Sessions temp database not found and copy failed: %v", copyErr))
				return []SessionInfo{}, nil
			}
		} else {
			panelLogger.Error(fmt.Sprintf("Failed to stat sessions temp database: %s", sessionsTempDBPath), err)
			return nil, fmt.Errorf("failed to access sessions temp database: %w", err)
		}
	}

	// Open database in ReadOnly mode ONLY (doesn't lock the file)
	// This is safe because the server keeps the database open for writing
	// and we only need to read it
	// Use longer timeout for Linux since the file is locked by the server
	// bbolt ReadOnly mode should work even when file is locked by another process
	timeout := 10 * time.Second
	if runtime.GOOS == "linux" {
		timeout = 120 * time.Second // Very long timeout for Linux when file is locked
	}
	
	// Always use ReadOnly mode for reading from temp file (doesn't lock the file)
	// Use NoGrowSync and NoSync for better compatibility
	// File mode 0444 (read-only) is safe for ReadOnly access
	db, err := bbolt.Open(sessionsTempDBPath, 0444, &bbolt.Options{
		Timeout:     timeout,
		ReadOnly:    true, // ReadOnly mode doesn't lock the file - allows concurrent access
		NoGrowSync:  true, // Don't sync on grow - helps with locked files
		NoSync:      true, // Don't sync - helps with ReadOnly on locked files
		MmapFlags:   0,    // Use default mmap flags
	})
	if err != nil {
		panelLogger.Error(fmt.Sprintf("Failed to open sessions temp database in ReadOnly mode: %s", sessionsTempDBPath), err)
		// Check if it's a permission issue
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied: check file permissions for %s (should be readable)", sessionsDBPath)
		}
		// Check if it's a timeout or lock issue
		errMsg := err.Error()
		if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "lock") || strings.Contains(errMsg, "resource") {
			// Try one more time with even longer timeout and wait
			panelLogger.Debug("Timeout/lock occurred, waiting and retrying with longer timeout...")
			time.Sleep(2 * time.Second) // Wait a bit for the lock to potentially release
			retryTimeout := 60 * time.Second
			if runtime.GOOS == "linux" {
				retryTimeout = 180 * time.Second // Very long timeout for retry
			}
			retryDb, retryErr := bbolt.Open(sessionsDBPath, 0444, &bbolt.Options{
				Timeout:     retryTimeout,
				ReadOnly:    true,
				NoGrowSync:  true,
				NoSync:      true,
				MmapFlags:   0,
			})
			if retryErr != nil {
				panelLogger.Error("Failed to open database after retry", retryErr)
				return nil, fmt.Errorf("database timeout: file is locked by server process (Abdal_SSH). ReadOnly mode cannot access locked write database. Please try again in a few moments or check if server is running")
			}
			db = retryDb
			err = nil
		} else {
			return nil, fmt.Errorf("failed to open database in ReadOnly mode: %w", err)
		}
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			panelLogger.Error("Failed to close sessions database", closeErr)
		}
	}()

	// Use View transaction for reading (read-only)
	err = db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(sessionsBucket))
		if bucket == nil {
			// Bucket doesn't exist, return empty list
			panelLogger.Debug("Sessions bucket does not exist in database")
			return nil
		}

		// Use cursor for better performance and to avoid locking issues
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			var session SessionInfo
			if err := json.Unmarshal(v, &session); err != nil {
				// Skip invalid entries
				panelLogger.Warning(fmt.Sprintf("Failed to parse session entry: %v", err))
				continue
			}
			// Only include non-revoked sessions
			if !session.Revoked {
				sessions = append(sessions, session)
			}
		}
		return nil
	})

	if err != nil {
		panelLogger.Error("Failed to read sessions from database", err)
		return nil, err
	}

	panelLogger.Debug(fmt.Sprintf("Loaded %d active sessions from database", len(sessions)))
	return sessions, nil
}

// getActiveSessionsCount returns the count of active (non-revoked) sessions
func getActiveSessionsCount() int {
	sessions, err := loadAllSessions()
	if err != nil {
		panelLogger.Error("Failed to get active sessions count", err)
		return 0
	}
	return len(sessions)
}

// deleteSession deletes a session from the database by marking it as revoked
// This function stops the server service, deletes the session, then starts the service again
func deleteSession(sessionID string) error {
	panelLogger.Info(fmt.Sprintf("Starting session deletion process for: %s", sessionID))

	// Step 1: Stop the server service to unlock the database
	panelLogger.Info("Stopping server service to unlock database...")
	if err := stopService(); err != nil {
		panelLogger.Error("Failed to stop server service", err)
		return fmt.Errorf("failed to stop server service: %w", err)
	}

	// Wait a bit for the service to fully stop and release the database lock
	time.Sleep(2 * time.Second)

	// Step 2: Delete the session from the database
	panelLogger.Info(fmt.Sprintf("Deleting session: %s", sessionID))
	
	// Ensure directory exists
	dbDir := filepath.Dir(sessionsDBPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		panelLogger.Error(fmt.Sprintf("Failed to create sessions directory: %s", dbDir), err)
		// Try to start service again even if directory creation failed
		startService()
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Open database with timeout (should work now as service is stopped)
	timeout := 10 * time.Second
	if runtime.GOOS == "linux" {
		timeout = 15 * time.Second
	}

	db, err := bbolt.Open(sessionsDBPath, 0600, &bbolt.Options{
		Timeout:     timeout,
		ReadOnly:    false, // Need write access for deletion
		NoGrowSync:  false,
		NoSync:      false,
	})
	if err != nil {
		panelLogger.Error(fmt.Sprintf("Failed to open sessions database for deletion: %s", sessionsDBPath), err)
		// Try to start service again even if database open failed
		startService()
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: check file permissions for %s", sessionsDBPath)
		}
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			panelLogger.Error("Failed to close sessions database after deletion", closeErr)
		}
	}()

	// Delete the session
	err = db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(sessionsBucket))
		if err != nil {
			return err
		}

		// Get session data
		sessionData := bucket.Get([]byte(sessionID))
		if sessionData == nil {
			return fmt.Errorf("session not found")
		}

		// Parse session to mark as revoked instead of deleting
		var session SessionInfo
		if err := json.Unmarshal(sessionData, &session); err != nil {
			return err
		}

		// Mark as revoked
		session.Revoked = true
		updatedData, err := json.Marshal(session)
		if err != nil {
			return err
		}

		// Update in database
		return bucket.Put([]byte(sessionID), updatedData)
	})

	if err != nil {
		panelLogger.Error(fmt.Sprintf("Failed to delete session: %s", sessionID), err)
		// Try to start service again even if deletion failed
		startService()
		return err
	}

	panelLogger.Info(fmt.Sprintf("Session %s successfully marked as revoked", sessionID))

	// Step 3: Start the server service again
	panelLogger.Info("Starting server service again...")
	if err := startService(); err != nil {
		panelLogger.Error("Failed to start server service after session deletion", err)
		return fmt.Errorf("session deleted but failed to start server service: %w", err)
	}

	panelLogger.Info(fmt.Sprintf("Session deletion completed successfully for: %s", sessionID))
	return nil
}

func getDefaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "/bin/bash"
}

// Logger handles logging to file
type Logger struct {
	enabled bool
	logFile *os.File
	logger  *log.Logger
	mu      sync.Mutex
}

// NewLogger creates a new logger instance
func NewLogger(enabled bool) *Logger {
	l := &Logger{
		enabled: enabled,
	}

	if !enabled {
		return l
	}

	// Open log file (append mode)
	logFile, err := os.OpenFile("abdal-4iproto-panel.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If can't open file, disable logging
		log.Printf("Warning: Could not open log file: %v", err)
		l.enabled = false
		return l
	}

	l.logFile = logFile
	// Create multi-writer: write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	l.logger = log.New(multiWriter, "", 0)

	return l
}

// log formats and writes log entry
func (l *Logger) log(level, message string) {
	if !l.enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)
	
	if l.logger != nil {
		l.logger.Println(formattedMessage)
	}
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log("INFO", message)
}

// Error logs an error message
func (l *Logger) Error(message string, err error) {
	if err != nil {
		l.log("ERROR", fmt.Sprintf("%s: %v", message, err))
	} else {
		l.log("ERROR", message)
	}
}

// Warning logs a warning message
func (l *Logger) Warning(message string) {
	l.log("WARNING", message)
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log("DEBUG", message)
}

// Request logs an HTTP request
func (l *Logger) Request(method, path, remoteAddr string, statusCode int, duration time.Duration) {
	l.log("REQUEST", fmt.Sprintf("%s %s | IP: %s | Status: %d | Duration: %v", 
		method, path, remoteAddr, statusCode, duration))
}

// Close closes the log file
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// loadPanelConfig loads panel configuration from JSON file
func loadPanelConfig() (*PanelConfig, error) {
	data, err := os.ReadFile(panelConfigFile)
	if err != nil {
		return nil, err
	}
	var config PanelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// savePanelConfig saves panel configuration to JSON file
func savePanelConfig(config *PanelConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(panelConfigFile, data, 0644)
}

// authMiddleware checks if user is authenticated
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for session cookie
		cookie, err := r.Cookie("panel_session")
		if err != nil {
			// Redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Verify session (simple check - in production use proper session management)
		if cookie.Value != "authenticated" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}

// getClientIP extracts the real client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// isIPBlocked checks if an IP is in the blocked list
func isIPBlocked(ip string) bool {
	for _, blockedIP := range panelConfig.BlockedIPs {
		if blockedIP == ip {
			return true
		}
	}
	return false
}

// checkLoginAttempts checks if IP has exceeded login attempts
func checkLoginAttempts(ip string) (bool, int, time.Time) {
	loginAttemptsMutex.RLock()
	defer loginAttemptsMutex.RUnlock()
	
	attempt, exists := loginAttempts[ip]
	if !exists {
		return false, 0, time.Time{}
	}
	
	// Check if blocking period has expired
	if attempt.Blocked && time.Now().Before(attempt.BlockedUntil) {
		return true, attempt.Attempts, attempt.BlockedUntil
	}
	
	// Reset if blocking period expired
	if attempt.Blocked && time.Now().After(attempt.BlockedUntil) {
		loginAttemptsMutex.RUnlock()
		loginAttemptsMutex.Lock()
		delete(loginAttempts, ip)
		loginAttemptsMutex.Unlock()
		loginAttemptsMutex.RLock()
		return false, 0, time.Time{}
	}
	
	// Check if within time window
	window := time.Duration(panelConfig.LoginAttemptWindow) * time.Second
	if time.Since(attempt.LastAttempt) > window {
		// Reset attempts after window expired
		loginAttemptsMutex.RUnlock()
		loginAttemptsMutex.Lock()
		delete(loginAttempts, ip)
		loginAttemptsMutex.Unlock()
		loginAttemptsMutex.RLock()
		return false, 0, time.Time{}
	}
	
	return false, attempt.Attempts, time.Time{}
}

// recordFailedLogin records a failed login attempt
func recordFailedLogin(ip string) {
	loginAttemptsMutex.Lock()
	defer loginAttemptsMutex.Unlock()
	
	attempt, exists := loginAttempts[ip]
	if !exists {
		attempt = &LoginAttempt{
			IP:        ip,
			Attempts:  0,
			LastAttempt: time.Now(),
			Blocked:   false,
		}
	}
	
	attempt.Attempts++
	attempt.LastAttempt = time.Now()
	
	// Block if exceeded max attempts
	if attempt.Attempts >= panelConfig.MaxLoginAttempts {
		attempt.Blocked = true
		attempt.BlockedUntil = time.Now().Add(time.Duration(panelConfig.BlockDuration) * time.Second)
		panelLogger.Warning(fmt.Sprintf("IP %s blocked due to %d failed login attempts. Blocked for %d seconds until %v", 
			ip, attempt.Attempts, panelConfig.BlockDuration, attempt.BlockedUntil))
		// Add to blocked IPs list
		if !isIPBlocked(ip) {
			panelConfig.BlockedIPs = append(panelConfig.BlockedIPs, ip)
			savePanelConfig(panelConfig)
		}
	}
	
	loginAttempts[ip] = attempt
}

// clearLoginAttempts clears login attempts for an IP (on successful login)
func clearLoginAttempts(ip string) {
	loginAttemptsMutex.Lock()
	defer loginAttemptsMutex.Unlock()
	delete(loginAttempts, ip)
}

// loginHandler handles login requests
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, r, "login.html", nil)
		return
	}

	if r.Method == http.MethodPost {
		clientIP := getClientIP(r)
		
		// Check if IP is blocked
		if isIPBlocked(clientIP) {
			panelLogger.Warning(fmt.Sprintf("Blocked IP attempted to login: %s", clientIP))
			data := map[string]interface{}{
				"Error": fmt.Sprintf("Your IP address (%s) is blocked. Please contact administrator.", clientIP),
			}
			renderTemplate(w, r, "login.html", data)
			return
		}
		
		// Check login attempts
		isBlocked, attempts, blockedUntil := checkLoginAttempts(clientIP)
		if isBlocked {
			remaining := time.Until(blockedUntil)
			panelLogger.Warning(fmt.Sprintf("IP %s is temporarily blocked. Attempts: %d. Remaining: %v", 
				clientIP, attempts, remaining))
			data := map[string]interface{}{
				"Error": fmt.Sprintf("Too many failed login attempts. Please try again after %v", remaining.Round(time.Second)),
			}
			renderTemplate(w, r, "login.html", data)
			return
		}
		
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == panelConfig.Username && password == panelConfig.Password {
			// Clear login attempts on successful login
			clearLoginAttempts(clientIP)
			
			// Set session cookie
			cookie := http.Cookie{
				Name:     "panel_session",
				Value:    "authenticated",
				Path:     "/",
				MaxAge:   3600 * 24, // 24 hours
				HttpOnly: true,
			}
			http.SetCookie(w, &cookie)
			panelLogger.Info(fmt.Sprintf("Successful login from IP: %s", clientIP))
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// Login failed - record attempt
		recordFailedLogin(clientIP)
		_, attempts, _ = checkLoginAttempts(clientIP)
		remaining := panelConfig.MaxLoginAttempts - attempts
		
		panelLogger.Warning(fmt.Sprintf("Failed login attempt from IP: %s (Username: %s). Attempts: %d/%d", 
			clientIP, username, attempts, panelConfig.MaxLoginAttempts))
		
		errorMsg := "Invalid username or password"
		if remaining > 0 {
			errorMsg = fmt.Sprintf("Invalid username or password. %d attempts remaining.", remaining)
		} else {
			errorMsg = fmt.Sprintf("Too many failed attempts. IP blocked for %d seconds.", panelConfig.BlockDuration)
		}
		
		data := map[string]interface{}{
			"Error": errorMsg,
		}
		renderTemplate(w, r, "login.html", data)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// logoutHandler handles logout requests
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	panelLogger.Info(fmt.Sprintf("User logged out from IP: %s", r.RemoteAddr))
	cookie := http.Cookie{
		Name:     "panel_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// stopService stops the service based on OS
func stopService() error {
	if runtime.GOOS == "windows" {
		panelLogger.Info(fmt.Sprintf("Stopping Windows service: %s", windowsServerService))
		cmd := exec.Command("sc", "stop", windowsServerService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to stop service: %s", windowsServerService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Service stopped successfully: %s", windowsServerService))
		return nil
	} else {
		panelLogger.Info(fmt.Sprintf("Stopping Linux service: %s", linuxServerService))
		cmd := exec.Command("systemctl", "stop", linuxServerService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to stop service: %s", linuxServerService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Service stopped successfully: %s", linuxServerService))
		return nil
	}
}

// startService starts the service based on OS
func startService() error {
	if runtime.GOOS == "windows" {
		panelLogger.Info(fmt.Sprintf("Starting Windows service: %s", windowsServerService))
		cmd := exec.Command("sc", "start", windowsServerService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to start service: %s", windowsServerService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Service started successfully: %s", windowsServerService))
		return nil
	} else {
		panelLogger.Info(fmt.Sprintf("Starting Linux service: %s", linuxServerService))
		cmd := exec.Command("systemctl", "start", linuxServerService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to start service: %s", linuxServerService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Service started successfully: %s", linuxServerService))
		return nil
	}
}

// apiRestartServerServiceHandler handles restart server service API requests
func apiRestartServerServiceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Restart server service in a goroutine to avoid blocking
	go func() {
		restartService()
		panelLogger.Info("Server service restart triggered via API")
	}()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Server service restart initiated"})
}

// apiRestartPanelServiceHandler handles restart panel service API requests
func apiRestartPanelServiceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Restart panel service in a goroutine to avoid blocking
	go func() {
		restartPanelService()
		panelLogger.Info("Panel service restart triggered via API")
	}()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Panel service restart initiated"})
}

// restartService restarts the service based on OS
func restartService() {
	if err := stopService(); err != nil {
		// Log error but continue
		panelLogger.Error("Failed to stop service during restart", err)
	}
	time.Sleep(1 * time.Second)
	if err := startService(); err != nil {
		// Log error but continue
		panelLogger.Error("Failed to start service during restart", err)
	} else {
		if runtime.GOOS == "windows" {
			panelLogger.Info(fmt.Sprintf("Service restarted successfully: %s", windowsServerService))
		} else {
			panelLogger.Info(fmt.Sprintf("Service restarted successfully: %s", linuxServerService))
		}
	}
}

// stopPanelService stops the panel service based on OS
func stopPanelService() error {
	if runtime.GOOS == "windows" {
		panelLogger.Info(fmt.Sprintf("Stopping Windows panel service: %s", windowsPanelService))
		cmd := exec.Command("sc", "stop", windowsPanelService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to stop panel service: %s", windowsPanelService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Panel service stopped successfully: %s", windowsPanelService))
		return nil
	} else {
		panelLogger.Info(fmt.Sprintf("Stopping Linux panel service: %s", linuxPanelService))
		cmd := exec.Command("systemctl", "stop", linuxPanelService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to stop panel service: %s", linuxPanelService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Panel service stopped successfully: %s", linuxPanelService))
		return nil
	}
}

// startPanelService starts the panel service based on OS
func startPanelService() error {
	if runtime.GOOS == "windows" {
		panelLogger.Info(fmt.Sprintf("Starting Windows panel service: %s", windowsPanelService))
		cmd := exec.Command("sc", "start", windowsPanelService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to start panel service: %s", windowsPanelService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Panel service started successfully: %s", windowsPanelService))
		return nil
	} else {
		panelLogger.Info(fmt.Sprintf("Starting Linux panel service: %s", linuxPanelService))
		cmd := exec.Command("systemctl", "start", linuxPanelService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to start panel service: %s", linuxPanelService), err)
			return err
		}
		panelLogger.Info(fmt.Sprintf("Panel service started successfully: %s", linuxPanelService))
		return nil
	}
}

// restartPanelService restarts the panel service based on OS
// Note: This function should be called from within the service itself,
// so it uses systemctl restart which is atomic and safe
func restartPanelService() {
	if runtime.GOOS == "windows" {
		panelLogger.Info(fmt.Sprintf("Restarting Windows panel service: %s", windowsPanelService))
		// On Windows, we need to stop and start separately
		// But since we're running as a service, we can use net stop/start
		// Or better: use sc command with timeout
		cmd := exec.Command("sc", "stop", windowsPanelService)
		if err := cmd.Run(); err != nil {
			panelLogger.Warning(fmt.Sprintf("Failed to stop panel service (may already be stopped): %v", err))
		}
		time.Sleep(2 * time.Second)
		cmd = exec.Command("sc", "start", windowsPanelService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to start panel service: %s", windowsPanelService), err)
			return
		}
		panelLogger.Info(fmt.Sprintf("Panel service restarted successfully: %s", windowsPanelService))
	} else {
		// On Linux, use systemctl restart which is atomic and safe
		// It stops and starts the service in one command without issues
		panelLogger.Info(fmt.Sprintf("Restarting Linux panel service: %s", linuxPanelService))
		cmd := exec.Command("systemctl", "restart", linuxPanelService)
		if err := cmd.Run(); err != nil {
			panelLogger.Error(fmt.Sprintf("Failed to restart panel service: %s", linuxPanelService), err)
			return
		}
		panelLogger.Info(fmt.Sprintf("Panel service restarted successfully: %s", linuxPanelService))
	}
}

