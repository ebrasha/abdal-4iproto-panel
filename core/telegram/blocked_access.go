/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : blocked_access.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 23:04:00
 * Description  : Telegram bot flows for browsing per user blocked
 *                access log files.
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

const blockedAccessTailCount = 20

// handleBlockedAccessCmd opens the blocked access list via command
func (tb *Bot) handleBlockedAccessCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderBlockedAccessList(ctx, update.Message.Chat.ID, 0, 0, lang)
}

// handleBlockedAccessCallback dispatches the blocked access callbacks
func (tb *Bot) handleBlockedAccessCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID
	parts := strings.SplitN(cq.Data, ":", 3) // ba:action[:arg]
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
		tb.renderBlockedAccessList(ctx, chatID, msgID, page, lang)
	case "view":
		if len(parts) < 3 {
			return
		}
		tb.renderBlockedAccessLog(ctx, chatID, parts[2], lang)
	}
}

func (tb *Bot) renderBlockedAccessList(ctx context.Context, chatID int64, msgID int, page int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	users, err := tb.svc.ListBlockedAccessLogs()
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
		rows = append(rows, kb(btn("➖ "+t("tg_blocked_access_empty"), CBNoop)))
	}
	for _, u := range users[start:end] {
		rows = append(rows, kb(btn("🛑 "+u, fmt.Sprintf("%s:view:%s", CBBlocked, u))))
	}
	if nav := PaginationRow(tb.svc, lang, CBBlocked+":list", page, totalPages); nav != nil {
		rows = append(rows, nav)
	}
	rows = append(rows, kb(btn("🏠 "+t("tg_btn_main_menu"), CBMenu)))

	text := fmt.Sprintf("🛑 <b>%s</b>\n%s %d / %d", t("blocked_access"), t("tg_page"), page+1, totalPages)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, markup)
		return
	}
	tb.sendText(ctx, chatID, text, markup)
}

func (tb *Bot) renderBlockedAccessLog(ctx context.Context, chatID int64, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	entries, err := tb.svc.ReadBlockedAccessLog(username, blockedAccessTailCount)
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_logs")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🛑 <b>%s</b>: <code>%s</code>\n", t("blocked_access"), EscapeHTML(username)))
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
				btn("🔙 "+t("tg_btn_back"), CBBlocked+":list:0"),
				btn("🏠 "+t("tg_btn_main_menu"), CBMenu),
			),
		},
	}
	tb.sendText(ctx, chatID, sb.String(), markup)
}
