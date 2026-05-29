/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : sessions.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 23:07:00
 * Description  : Telegram bot flows for listing and revoking active
 *                sessions stored in the bbolt sessions database.
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package telegram

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// sessionCallbackToken stores the full session ID behind a short deterministic
// token because Telegram callback_data is limited to 64 bytes.
func (tb *Bot) sessionCallbackToken(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	token := hex.EncodeToString(sum[:8])

	tb.sessionTokenMu.Lock()
	if tb.sessionTokens == nil {
		tb.sessionTokens = make(map[string]string)
	}
	tb.sessionTokens[token] = sessionID
	tb.sessionTokenMu.Unlock()

	return token
}

// resolveSessionCallbackToken maps the short callback token back to the real
// session ID. The fallback preserves compatibility with any old callback data
// that may still contain the full session ID.
func (tb *Bot) resolveSessionCallbackToken(token string) string {
	tb.sessionTokenMu.RLock()
	sessionID := tb.sessionTokens[token]
	tb.sessionTokenMu.RUnlock()
	if sessionID != "" {
		return sessionID
	}
	return token
}

// handleSessionsCmd opens the sessions list via command
func (tb *Bot) handleSessionsCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderSessionsList(ctx, update.Message.Chat.ID, 0, 0, lang)
}

// handleSessionsCallback dispatches session callbacks
func (tb *Bot) handleSessionsCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID
	parts := strings.SplitN(cq.Data, ":", 3) // se:action[:arg]
	if len(parts) < 2 {
		return
	}
	switch parts[1] {
	case "list":
		page := 0
		if len(parts) >= 3 {
			if p, err := strconv.Atoi(parts[2]); err == nil {
				page = p
			}
		}
		tb.renderSessionsList(ctx, chatID, msgID, page, lang)
	case "del":
		if len(parts) < 3 {
			return
		}
		tb.renderSessionDeleteConfirm(ctx, chatID, msgID, parts[2], lang)
	case "del_yes":
		if len(parts) < 3 {
			return
		}
		tb.performDeleteSession(ctx, chatID, msgID, parts[2], lang)
	}
}

// renderSessionsList prints active sessions with delete actions
func (tb *Bot) renderSessionsList(ctx context.Context, chatID int64, msgID int, page int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	sessions, err := tb.svc.LoadSessions()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_sessions")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	sort.Slice(sessions, func(i, j int) bool { return sessions[i].LastSeen > sessions[j].LastSeen })

	totalPages := (len(sessions) + PageSize - 1) / PageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	start := page * PageSize
	end := start + PageSize
	if end > len(sessions) {
		end = len(sessions)
	}

	var rows [][]models.InlineKeyboardButton
	if len(sessions) == 0 {
		rows = append(rows, kb(btn("➖ "+t("tg_sessions_empty"), CBNoop)))
	}
	for _, s := range sessions[start:end] {
		shortID := TruncateMiddle(s.SessionID, 14)
		label := fmt.Sprintf("🔗 %s | %s", s.Username, shortID)
		sessionToken := tb.sessionCallbackToken(s.SessionID)
		rows = append(rows, kb(
			btn(label, CBNoop),
			btn("🗑️", fmt.Sprintf("%s:del:%s", CBSessions, sessionToken)),
		))
	}
	if nav := PaginationRow(tb.svc, lang, CBSessions+":list", page, totalPages); nav != nil {
		rows = append(rows, nav)
	}
	rows = append(rows, kb(btn("🏠 "+t("tg_btn_main_menu"), CBMenu)))

	text := fmt.Sprintf("🔗 <b>%s</b>\n%s %d / %d", t("sessions"), t("tg_page"), page+1, totalPages)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, markup)
		return
	}
	tb.sendText(ctx, chatID, text, markup)
}

// renderSessionDeleteConfirm asks before revoking a session
func (tb *Bot) renderSessionDeleteConfirm(ctx context.Context, chatID int64, msgID int, sessionToken, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	sessionID := tb.resolveSessionCallbackToken(sessionToken)
	text := fmt.Sprintf("⚠️ %s\n<code>%s</code>", t("tg_confirm_delete_session"), EscapeHTML(sessionID))
	yesData := fmt.Sprintf("%s:del_yes:%s", CBSessions, sessionToken)
	noData := fmt.Sprintf("%s:list:0", CBSessions)
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, ConfirmKeyboard(tb.svc, lang, yesData, noData))
		return
	}
	tb.sendText(ctx, chatID, text, ConfirmKeyboard(tb.svc, lang, yesData, noData))
}

// performDeleteSession revokes the chosen session
func (tb *Bot) performDeleteSession(ctx context.Context, chatID int64, msgID int, sessionToken, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	sessionID := tb.resolveSessionCallbackToken(sessionToken)
	tb.sendText(ctx, chatID, "⏳ "+t("tg_session_deleting"), nil)
	if err := tb.svc.DeleteSession(sessionID); err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_deleting_session")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	tb.sendText(ctx, chatID, "✅ "+t("tg_session_deleted"), nil)
	tb.renderSessionsList(ctx, chatID, msgID, 0, lang)
}
