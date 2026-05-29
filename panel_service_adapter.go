/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : panel_service_adapter.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 23:12:00
 * Description  : Adapter that implements the Telegram bot
 *                PanelService interface by delegating to the existing
 *                panel storage helpers and service control routines.
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tg "abdal-4iproto-panel/core/telegram"
)

// Tuning constants for adapter side concurrency primitives. Keeping these
// generous makes sure the Telegram bot is never throttled by the host panel.
const (
	// adapterLogBufferSize is the capacity of the async log channel. When the
	// panel logger is briefly busy the bot can keep producing log entries
	// without blocking. If the buffer fills up the oldest entries are dropped
	// silently in favour of keeping the bot responsive.
	adapterLogBufferSize = 1024

	// restartDebounceDelay coalesces a burst of mutations (e.g. adding several
	// users back to back from the Telegram bot) into a single Abdal_SSH
	// service restart. Without it, every request would queue its own
	// stop/start cycle and starve other operations.
	restartDebounceDelay = 1500 * time.Millisecond
)

// adapterLogEntry is a single log record awaiting delivery to panelLogger.
// It is intentionally small so the buffered channel costs almost nothing.
type adapterLogEntry struct {
	level   string
	message string
	err     error
}

// panelServiceAdapter glues the main package to the telegram package.
// All write operations also trigger the same service restart logic the
// HTTP handlers use, so the Abdal_SSH server reloads its state.
//
// Concurrency design:
//   - logCh is a buffered, fire-and-forget channel served by a single
//     goroutine. Bot handlers never block on the panel logger mutex.
//   - restartSignal triggers a debounced server restart so multiple rapid
//     mutations collapse into a single sc/systemctl stop+start cycle.
type panelServiceAdapter struct {
	logCh         chan adapterLogEntry
	restartSignal chan struct{}
	startOnce     sync.Once
}

// newPanelServiceAdapter builds an adapter and spins up its background
// goroutines (log writer and debounced restart trigger).
func newPanelServiceAdapter() *panelServiceAdapter {
	a := &panelServiceAdapter{
		logCh:         make(chan adapterLogEntry, adapterLogBufferSize),
		restartSignal: make(chan struct{}, 1),
	}
	a.startOnce.Do(func() {
		go a.runLogWorker()
		go a.runRestartDebouncer()
	})
	return a
}

// runLogWorker forwards bot log entries to the shared panel logger. Running in
// a dedicated goroutine isolates the bot from any contention on Logger.mu and
// from slow disk writes inside the panel log file.
func (a *panelServiceAdapter) runLogWorker() {
	for entry := range a.logCh {
		if panelLogger == nil {
			continue
		}
		switch entry.level {
		case "INFO":
			panelLogger.Info(entry.message)
		case "WARNING":
			panelLogger.Warning(entry.message)
		case "ERROR":
			panelLogger.Error(entry.message, entry.err)
		}
	}
}

// runRestartDebouncer waits for a restart request and then sleeps for the
// debounce window to absorb any follow-up requests, so several mutations
// performed within a short period collapse into a single restartService call.
func (a *panelServiceAdapter) runRestartDebouncer() {
	for range a.restartSignal {
		// Drain any extra signals that may already be pending so they do not
		// trigger a second restart immediately after this one.
		timer := time.NewTimer(restartDebounceDelay)
	drain:
		for {
			select {
			case <-a.restartSignal:
				// Reset the timer to extend the debounce window
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(restartDebounceDelay)
			case <-timer.C:
				break drain
			}
		}
		restartService()
	}
}

// enqueueLog pushes a log entry without ever blocking the caller. When the
// buffer is saturated the entry is dropped, which is safe for our use case
// (visibility logs only, no data correctness implication).
func (a *panelServiceAdapter) enqueueLog(level, message string, err error) {
	select {
	case a.logCh <- adapterLogEntry{level: level, message: message, err: err}:
	default:
	}
}

// triggerRestart schedules a debounced server service restart. Non-blocking:
// if a restart is already queued, the signal is dropped.
func (a *panelServiceAdapter) triggerRestart() {
	select {
	case a.restartSignal <- struct{}{}:
	default:
	}
}

// telegramBotConfig returns the bot config translated to the bot package types
func telegramBotConfig() tg.BotConfig {
	if panelConfig == nil {
		return tg.BotConfig{}
	}
	admins := make([]int64, len(panelConfig.TelegramBot.Admins))
	copy(admins, panelConfig.TelegramBot.Admins)
	return tg.BotConfig{
		Enabled: panelConfig.TelegramBot.Enabled,
		Token:   panelConfig.TelegramBot.Token,
		Admins:  admins,
	}
}

// ----- Translations -------------------------------------------------------

// Translate proxies to the existing panel translation helper
func (a *panelServiceAdapter) Translate(lang, key string) string {
	return getTranslation(lang, key)
}

// ----- Logging ------------------------------------------------------------
// All logger methods enqueue into the buffered logCh so a contended panel
// Logger.mu can never slow down a Telegram handler.

// Info routes informational messages into the panel log
func (a *panelServiceAdapter) Info(msg string) {
	a.enqueueLog("INFO", "[Telegram] "+msg, nil)
}

// Warning routes warnings into the panel log
func (a *panelServiceAdapter) Warning(msg string) {
	a.enqueueLog("WARNING", "[Telegram] "+msg, nil)
}

// Error routes errors into the panel log
func (a *panelServiceAdapter) Error(msg string, err error) {
	a.enqueueLog("ERROR", "[Telegram] "+msg, err)
}

// ----- Users --------------------------------------------------------------

// LoadUsers returns all configured users converted to the bot type
func (a *panelServiceAdapter) LoadUsers() ([]tg.User, error) {
	users, err := loadUsers()
	if err != nil {
		return nil, err
	}
	out := make([]tg.User, 0, len(users))
	for _, u := range users {
		out = append(out, userToTG(u))
	}
	return out, nil
}

// GetUser returns a single user by name
func (a *panelServiceAdapter) GetUser(username string) (*tg.User, error) {
	users, err := loadUsers()
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.Username == username {
			converted := userToTG(u)
			return &converted, nil
		}
	}
	return nil, nil
}

// AddUser appends a new user to the users file and restarts the server service
func (a *panelServiceAdapter) AddUser(user tg.User) error {
	users, err := loadUsers()
	if err != nil {
		return err
	}
	for _, u := range users {
		if u.Username == user.Username {
			return fmt.Errorf("user %q already exists", user.Username)
		}
	}
	users = append(users, userFromTG(user))
	if err := saveUsers(users); err != nil {
		return err
	}
	a.triggerRestart()
	return nil
}

// UpdateUser replaces an existing user and restarts the server service
func (a *panelServiceAdapter) UpdateUser(user tg.User) error {
	users, err := loadUsers()
	if err != nil {
		return err
	}
	found := false
	for i, u := range users {
		if u.Username == user.Username {
			users[i] = userFromTG(user)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("user %q not found", user.Username)
	}
	if err := saveUsers(users); err != nil {
		return err
	}
	a.triggerRestart()
	return nil
}

// DeleteUser removes a user and restarts the server service
func (a *panelServiceAdapter) DeleteUser(username string) error {
	users, err := loadUsers()
	if err != nil {
		return err
	}
	updated := make([]User, 0, len(users))
	found := false
	for _, u := range users {
		if u.Username == username {
			found = true
			continue
		}
		updated = append(updated, u)
	}
	if !found {
		return fmt.Errorf("user %q not found", username)
	}
	if err := saveUsers(updated); err != nil {
		return err
	}
	a.triggerRestart()
	return nil
}

// ----- Server configuration ----------------------------------------------

// LoadServerConfig returns the persisted server config
func (a *panelServiceAdapter) LoadServerConfig() (*tg.ServerConfig, error) {
	cfg, err := loadServerConfig()
	if err != nil {
		return nil, err
	}
	conv := serverConfigToTG(*cfg)
	return &conv, nil
}

// SaveServerConfig writes the new server config and restarts the server service
func (a *panelServiceAdapter) SaveServerConfig(cfg *tg.ServerConfig) error {
	if cfg == nil {
		return fmt.Errorf("server config is nil")
	}
	native := serverConfigFromTG(*cfg)
	if err := saveServerConfig(&native); err != nil {
		return err
	}
	a.triggerRestart()
	return nil
}

// ----- Blocked IPs -------------------------------------------------------

// LoadBlockedIPs returns the panel level blocked IPs
func (a *panelServiceAdapter) LoadBlockedIPs() ([]string, error) {
	if panelConfig == nil {
		return []string{}, nil
	}
	out := make([]string, len(panelConfig.BlockedIPs))
	copy(out, panelConfig.BlockedIPs)
	return out, nil
}

// SaveBlockedIPs replaces the panel level blocked IPs and persists the config
func (a *panelServiceAdapter) SaveBlockedIPs(ips []string) error {
	if panelConfig == nil {
		return fmt.Errorf("panel config not initialized")
	}
	panelConfig.BlockedIPs = ips
	return savePanelConfig(panelConfig)
}

// ----- Logs --------------------------------------------------------------

// ListUserLogs returns the usernames that have access log files
func (a *panelServiceAdapter) ListUserLogs() ([]string, error) {
	return getLogFiles()
}

// ReadUserLog returns the last N parsed entries from the user's access log
func (a *panelServiceAdapter) ReadUserLog(username string, lastN int) ([]tg.LogEntry, error) {
	return readParsedLog(filepath.Join(usersLogDir, username+".log"), lastN)
}

// ListBlockedAccessLogs returns the usernames that have blocked access log files
func (a *panelServiceAdapter) ListBlockedAccessLogs() ([]string, error) {
	return getBlockedAccessLogFiles()
}

// ReadBlockedAccessLog returns the last N parsed entries from the user's blocked access log
func (a *panelServiceAdapter) ReadBlockedAccessLog(username string, lastN int) ([]tg.LogEntry, error) {
	return readParsedLog(filepath.Join(blockedAccessDir, username+".log"), lastN)
}

// ----- Traffic -----------------------------------------------------------

// LoadTraffic returns traffic counters for a single user
func (a *panelServiceAdapter) LoadTraffic(username string) (*tg.TrafficData, error) {
	tr, err := loadTrafficData(username)
	if err != nil {
		return nil, err
	}
	if tr == nil {
		return nil, nil
	}
	conv := trafficToTG(*tr)
	return &conv, nil
}

// ----- Sessions ----------------------------------------------------------

// LoadSessions returns the active session list
func (a *panelServiceAdapter) LoadSessions() ([]tg.SessionInfo, error) {
	sessions, err := loadAllSessions()
	if err != nil {
		return nil, err
	}
	out := make([]tg.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sessionToTG(s))
	}
	return out, nil
}

// DeleteSession revokes a session by ID using the existing delete pipeline
func (a *panelServiceAdapter) DeleteSession(sessionID string) error {
	return deleteSession(sessionID)
}

// ----- Service control ---------------------------------------------------

// RestartServerService schedules a debounced restart of the Abdal 4iProto
// Server service. Returns immediately so the Telegram handler stays responsive.
func (a *panelServiceAdapter) RestartServerService() error {
	a.triggerRestart()
	return nil
}

// RestartPanelService restarts the panel service. Caller should expect the
// current process to die shortly after this call returns.
func (a *panelServiceAdapter) RestartPanelService() error {
	go restartPanelService()
	return nil
}

// ----- Conversion helpers ------------------------------------------------

func userToTG(u User) tg.User {
	return tg.User{
		Username:          u.Username,
		Password:          u.Password,
		Role:              u.Role,
		BlockedDomains:    append([]string(nil), u.BlockedDomains...),
		BlockedIPs:        append([]string(nil), u.BlockedIPs...),
		Log:               u.Log,
		MaxSessions:       u.MaxSessions,
		SessionTTLSeconds: u.SessionTTLSeconds,
		MaxSpeedKbps:      u.MaxSpeedKbps,
		MaxTotalMB:        u.MaxTotalMB,
	}
}

func userFromTG(u tg.User) User {
	return User{
		Username:          u.Username,
		Password:          u.Password,
		Role:              u.Role,
		BlockedDomains:    append([]string(nil), u.BlockedDomains...),
		BlockedIPs:        append([]string(nil), u.BlockedIPs...),
		Log:               u.Log,
		MaxSessions:       u.MaxSessions,
		SessionTTLSeconds: u.SessionTTLSeconds,
		MaxSpeedKbps:      u.MaxSpeedKbps,
		MaxTotalMB:        u.MaxTotalMB,
	}
}

func serverConfigToTG(c ServerConfig) tg.ServerConfig {
	return tg.ServerConfig{
		Ports:           append([]int(nil), c.Ports...),
		Shell:           c.Shell,
		MaxAuthAttempts: c.MaxAuthAttempts,
		ServerVersion:   c.ServerVersion,
		PrivateKeyFile:  c.PrivateKeyFile,
		PublicKeyFile:   c.PublicKeyFile,
	}
}

func serverConfigFromTG(c tg.ServerConfig) ServerConfig {
	return ServerConfig{
		Ports:           append([]int(nil), c.Ports...),
		Shell:           c.Shell,
		MaxAuthAttempts: c.MaxAuthAttempts,
		ServerVersion:   c.ServerVersion,
		PrivateKeyFile:  c.PrivateKeyFile,
		PublicKeyFile:   c.PublicKeyFile,
	}
}

func sessionToTG(s SessionInfo) tg.SessionInfo {
	return tg.SessionInfo{
		SessionID:     s.SessionID,
		Username:      s.Username,
		IP:            s.IP,
		ClientVersion: s.ClientVersion,
		CreatedAt:     s.CreatedAt,
		LastSeen:      s.LastSeen,
		Revoked:       s.Revoked,
	}
}

func trafficToTG(t TrafficData) tg.TrafficData {
	return tg.TrafficData{
		Username:           t.Username,
		IP:                 t.IP,
		LastBytesSent:      t.LastBytesSent,
		LastBytesReceived:  t.LastBytesReceived,
		LastBytesTotal:     t.LastBytesTotal,
		TotalBytesSent:     t.TotalBytesSent,
		TotalBytesReceived: t.TotalBytesReceived,
		TotalBytes:         t.TotalBytes,
		LastTimestamp:      t.LastTimestamp,
	}
}

// readParsedLog reads a log file and returns the last N parsed entries.
// Parsing matches the format produced by the Abdal_SSH server, e.g.
// "[📡 User Access] [2025-11-04 15:26:49] Target: ... | User IP: ...".
func readParsedLog(path string, lastN int) ([]tg.LogEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []tg.LogEntry{}, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	entries := make([]tg.LogEntry, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := tg.LogEntry{Raw: line}
		parts := strings.Split(line, "]")
		if len(parts) >= 3 {
			entry.Timestamp = strings.TrimSpace(strings.TrimPrefix(parts[1], "["))
			rest := strings.Join(parts[2:], "]")
			targetParts := strings.Split(rest, "|")
			if len(targetParts) >= 1 {
				entry.Target = strings.TrimSpace(strings.TrimPrefix(targetParts[0], "Target:"))
			}
			if len(targetParts) >= 2 {
				entry.UserIP = strings.TrimSpace(strings.TrimPrefix(targetParts[1], "User IP:"))
			}
		}
		entries = append(entries, entry)
	}
	if lastN > 0 && len(entries) > lastN {
		entries = entries[len(entries)-lastN:]
	}
	return entries, nil
}

// Ensure the adapter satisfies the telegram.PanelService interface at compile time.
var _ tg.PanelService = (*panelServiceAdapter)(nil)
