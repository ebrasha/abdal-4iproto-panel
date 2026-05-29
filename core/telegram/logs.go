/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : logs.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 23:02:00
 * Description  : Telegram bot flows for browsing per user access logs.
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
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const logsTailCount = 20

// handleLogsCmd opens the logs list via command
func (tb *Bot) handleLogsCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderLogsList(ctx, update.Message.Chat.ID, 0, 0, lang)
}

// handleLogsCallback dispatches log namespace callbacks
func (tb *Bot) handleLogsCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID
	parts := strings.SplitN(cq.Data, ":", 3) // lg:action[:arg]
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
		tb.renderLogsList(ctx, chatID, msgID, page, lang)
	case "view":
		if len(parts) < 3 {
			return
		}
		tb.renderUserLog(ctx, chatID, parts[2], lang)
	}
}

// renderLogsList shows users that have a log file
func (tb *Bot) renderLogsList(ctx context.Context, chatID int64, msgID int, page int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	users, err := tb.svc.ListUserLogs()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_logs")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	sort.Strings(users)
	totalPages := (len(users) + PageSize - 1) / PageSize
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
	if end > len(users) {
		end = len(users)
	}

	var rows [][]models.InlineKeyboardButton
	if len(users) == 0 {
		rows = append(rows, kb(btn("➖ "+t("tg_logs_empty"), CBNoop)))
	}
	for _, u := range users[start:end] {
		rows = append(rows, kb(btn("📋 "+u, fmt.Sprintf("%s:view:%s", CBLogs, u))))
	}
	if nav := PaginationRow(tb.svc, lang, CBLogs+":list", page, totalPages); nav != nil {
		rows = append(rows, nav)
	}
	rows = append(rows, kb(btn("🏠 "+t("tg_btn_main_menu"), CBMenu)))

	text := fmt.Sprintf("📋 <b>%s</b>\n%s %d / %d", t("logs"), t("tg_page"), page+1, totalPages)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, markup)
		return
	}
	tb.sendText(ctx, chatID, text, markup)
}

// renderUserLog prints the last entries of a user's access log
func (tb *Bot) renderUserLog(ctx context.Context, chatID int64, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	entries, err := tb.svc.ReadUserLog(username, logsTailCount)
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_logs")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>%s</b>: <code>%s</code>\n", t("logs"), EscapeHTML(username)))
	if len(entries) == 0 {
		sb.WriteString(t("tg_logs_no_entries"))
	} else {
		for _, e := range entries {
			line := fmt.Sprintf("<code>%s</code> | 🎯 %s | 🌐 %s",
				EscapeHTML(e.Timestamp), EscapeHTML(e.Target), EscapeHTML(e.UserIP))
			sb.WriteString(line + "\n")
		}
	}
	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(
				btn("🔙 "+t("tg_btn_back"), CBLogs+":list:0"),
				btn("🏠 "+t("tg_btn_main_menu"), CBMenu),
			),
		},
	}
	tb.sendText(ctx, chatID, sb.String(), markup)
}
