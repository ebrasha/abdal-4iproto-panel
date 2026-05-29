/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : utils.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:48:00
 * Description  : Pure utility helpers for the Telegram bot package:
 *                random generators, formatters and text helpers.
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package telegram

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/go-telegram/bot/models"
)

const (
	usernameAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	passwordAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// randomString returns a cryptographically random string of length n from the given alphabet.
// Falls back to a deterministic alphabet stride on rare RNG errors so callers never fail.
func randomString(alphabet string, n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// Fallback: use time-derived bytes so callers always get a string
		seed := time.Now().UnixNano()
		for i := range buf {
			buf[i] = byte(seed >> uint(8*(i%8)))
		}
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out)
}

// GenerateUsername returns an auto-generated username such as "tg_a3k9x2"
func GenerateUsername() string {
	return "tg_" + randomString(usernameAlphabet, 6)
}

// GeneratePassword returns an auto-generated alphanumeric password
func GeneratePassword() string {
	return randomString(passwordAlphabet, 12)
}

// FormatBytes prints a human-readable size for the given byte count
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FormatTimestamp converts a unix timestamp into a readable string
func FormatTimestamp(ts int64) string {
	if ts == 0 {
		return "-"
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

// SplitCSV splits a comma-separated string and trims whitespace from each entry,
// dropping empty values.
func SplitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "skip") || s == "-" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// EscapeHTML protects user-provided text from breaking HTML messages
func EscapeHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	)
	return r.Replace(s)
}

// UserIDFromUpdate extracts the originating user id from a Telegram update
func UserIDFromUpdate(u *models.Update) int64 {
	if u == nil {
		return 0
	}
	if u.CallbackQuery != nil {
		return u.CallbackQuery.From.ID
	}
	if u.Message != nil && u.Message.From != nil {
		return u.Message.From.ID
	}
	if u.EditedMessage != nil && u.EditedMessage.From != nil {
		return u.EditedMessage.From.ID
	}
	return 0
}

// ChatIDFromUpdate extracts the chat id where a reply should be sent
func ChatIDFromUpdate(u *models.Update) int64 {
	if u == nil {
		return 0
	}
	if u.CallbackQuery != nil && u.CallbackQuery.Message.Message != nil {
		return u.CallbackQuery.Message.Message.Chat.ID
	}
	if u.Message != nil {
		return u.Message.Chat.ID
	}
	if u.EditedMessage != nil {
		return u.EditedMessage.Chat.ID
	}
	return 0
}

// FormatUserSummary builds a multiline summary for a user record using i18n keys
func FormatUserSummary(svc PanelService, lang string, u *User) string {
	t := func(k string) string { return svc.Translate(lang, k) }
	var sb strings.Builder
	sb.WriteString("👤 <b>" + EscapeHTML(u.Username) + "</b>\n")
	sb.WriteString(fmt.Sprintf("🎭 %s: <code>%s</code>\n", t("role"), EscapeHTML(u.Role)))
	sb.WriteString(fmt.Sprintf("🔑 %s: <code>%s</code>\n", t("password"), EscapeHTML(u.Password)))
	sb.WriteString(fmt.Sprintf("📝 %s: <code>%s</code>\n", t("log"), EscapeHTML(u.Log)))
	sb.WriteString(fmt.Sprintf("👥 %s: <code>%d</code>\n", t("max_sessions"), u.MaxSessions))
	sb.WriteString(fmt.Sprintf("⏳ %s: <code>%d</code>\n", t("session_ttl_seconds"), u.SessionTTLSeconds))
	sb.WriteString(fmt.Sprintf("⚡ %s: <code>%d</code>\n", t("max_speed_kbps"), u.MaxSpeedKbps))
	sb.WriteString(fmt.Sprintf("📦 %s: <code>%d</code>\n", t("max_total_mb"), u.MaxTotalMB))
	if len(u.BlockedDomains) > 0 {
		sb.WriteString(fmt.Sprintf("🌐 %s: <code>%s</code>\n", t("blocked_domains"), EscapeHTML(strings.Join(u.BlockedDomains, ", "))))
	}
	if len(u.BlockedIPs) > 0 {
		sb.WriteString(fmt.Sprintf("🚫 %s: <code>%s</code>\n", t("blocked_ips"), EscapeHTML(strings.Join(u.BlockedIPs, ", "))))
	}
	return sb.String()
}

// FormatServerConfig builds a summary for the server config
func FormatServerConfig(svc PanelService, lang string, c *ServerConfig) string {
	t := func(k string) string { return svc.Translate(lang, k) }
	portsStr := make([]string, 0, len(c.Ports))
	for _, p := range c.Ports {
		portsStr = append(portsStr, fmt.Sprintf("%d", p))
	}
	var sb strings.Builder
	sb.WriteString("⚙️ <b>" + t("server_config") + "</b>\n\n")
	sb.WriteString(fmt.Sprintf("🔌 %s: <code>%s</code>\n", t("ports"), strings.Join(portsStr, ", ")))
	sb.WriteString(fmt.Sprintf("🐚 %s: <code>%s</code>\n", t("shell"), EscapeHTML(c.Shell)))
	sb.WriteString(fmt.Sprintf("🔁 %s: <code>%d</code>\n", t("max_auth_attempts"), c.MaxAuthAttempts))
	sb.WriteString(fmt.Sprintf("📛 %s: <code>%s</code>\n", t("server_version"), EscapeHTML(c.ServerVersion)))
	sb.WriteString(fmt.Sprintf("🔑 %s: <code>%s</code>\n", t("private_key_file"), EscapeHTML(c.PrivateKeyFile)))
	sb.WriteString(fmt.Sprintf("🗝️ %s: <code>%s</code>\n", t("public_key_file"), EscapeHTML(c.PublicKeyFile)))
	return sb.String()
}

// TruncateMiddle shortens a long string with an ellipsis in the middle
func TruncateMiddle(s string, max int) string {
	if max <= 3 || len(s) <= max {
		return s
	}
	half := (max - 3) / 2
	return s[:half] + "..." + s[len(s)-half:]
}
