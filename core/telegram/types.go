/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : types.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:45:00
 * Description  : Shared types and PanelService interface used by the
 *                Telegram bot integration. The bot package stays
 *                decoupled from the main package by talking to the
 *                panel through this interface only.
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package telegram

// BotConfig holds runtime settings for the Telegram bot
type BotConfig struct {
	Enabled bool
	Token   string
	Admins  []int64
}

// User mirrors the panel user entity that the bot can manage
type User struct {
	Username          string
	Password          string
	Role              string
	BlockedDomains    []string
	BlockedIPs        []string
	Log               string
	MaxSessions       int
	SessionTTLSeconds int
	MaxSpeedKbps      int
	MaxTotalMB        int
}

// ServerConfig mirrors the panel server configuration entity
type ServerConfig struct {
	Ports           []int
	Shell           string
	MaxAuthAttempts int
	ServerVersion   string
	PrivateKeyFile  string
	PublicKeyFile   string
}

// SessionInfo mirrors a single active session record
type SessionInfo struct {
	SessionID     string
	Username      string
	IP            string
	ClientVersion string
	CreatedAt     int64
	LastSeen      int64
	Revoked       bool
}

// TrafficData mirrors per user traffic accounting
type TrafficData struct {
	Username           string
	IP                 string
	LastBytesSent      int64
	LastBytesReceived  int64
	LastBytesTotal     int64
	TotalBytesSent     int64
	TotalBytesReceived int64
	TotalBytes         int64
	LastTimestamp      string
}

// LogEntry represents a single parsed log line shown to the operator
type LogEntry struct {
	Timestamp string
	Target    string
	UserIP    string
	Raw       string
}

// Logger is the minimal logging contract the bot expects from the panel
type Logger interface {
	Info(msg string)
	Warning(msg string)
	Error(msg string, err error)
}

// PanelService is the gateway the bot uses to read and mutate panel state.
// The host application (main package) must provide a concrete implementation
// that adapts panel storage and services to these methods.
type PanelService interface {
	// Translate returns a localized string for the given language and key.
	Translate(lang, key string) string

	// Users
	LoadUsers() ([]User, error)
	GetUser(username string) (*User, error)
	AddUser(user User) error
	UpdateUser(user User) error
	DeleteUser(username string) error

	// Server configuration
	LoadServerConfig() (*ServerConfig, error)
	SaveServerConfig(*ServerConfig) error

	// Blocked IPs
	LoadBlockedIPs() ([]string, error)
	SaveBlockedIPs([]string) error

	// User access logs
	ListUserLogs() ([]string, error)
	ReadUserLog(username string, lastN int) ([]LogEntry, error)

	// Blocked access logs
	ListBlockedAccessLogs() ([]string, error)
	ReadBlockedAccessLog(username string, lastN int) ([]LogEntry, error)

	// Traffic
	LoadTraffic(username string) (*TrafficData, error)

	// Sessions
	LoadSessions() ([]SessionInfo, error)
	DeleteSession(sessionID string) error

	// Service control
	RestartServerService() error
	RestartPanelService() error

	// Logging hook so the bot writes into the panel log
	Logger
}
