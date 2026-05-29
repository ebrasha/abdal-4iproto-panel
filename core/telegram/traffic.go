/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : traffic.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 23:05:00
 * Description  : Telegram bot flows for showing per user traffic usage.
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

// handleTrafficCmd opens the traffic list via command
func (tb *Bot) handleTrafficCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderTrafficList(ctx, update.Message.Chat.ID, 0, 0, lang)
}

// handleTrafficCallback dispatches traffic callbacks
func (tb *Bot) handleTrafficCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID
	parts := strings.SplitN(cq.Data, ":", 3) // tr:action[:arg]
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
		tb.renderTrafficList(ctx, chatID, msgID, page, lang)
	case "view":
		if len(parts) < 3 {
			return
		}
		tb.renderTrafficDetail(ctx, chatID, parts[2], lang)
	}
}

// renderTrafficList lists users with their total usage as buttons
func (tb *Bot) renderTrafficList(ctx context.Context, chatID int64, msgID int, page int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	users, err := tb.svc.LoadUsers()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_users")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })

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
		rows = append(rows, kb(btn("➖ "+t("tg_traffic_empty"), CBNoop)))
	}
	for _, u := range users[start:end] {
		tr, _ := tb.svc.LoadTraffic(u.Username)
		var label string
		if tr == nil {
			label = fmt.Sprintf("📊 %s | %s", u.Username, t("tg_no_data"))
		} else {
			label = fmt.Sprintf("📊 %s | %s", u.Username, FormatBytes(tr.TotalBytes))
		}
		rows = append(rows, kb(btn(label, fmt.Sprintf("%s:view:%s", CBTraffic, u.Username))))
	}
	if nav := PaginationRow(tb.svc, lang, CBTraffic+":list", page, totalPages); nav != nil {
		rows = append(rows, nav)
	}
	rows = append(rows, kb(btn("🏠 "+t("tg_btn_main_menu"), CBMenu)))

	text := fmt.Sprintf("📊 <b>%s</b>\n%s %d / %d", t("traffic"), t("tg_page"), page+1, totalPages)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, markup)
		return
	}
	tb.sendText(ctx, chatID, text, markup)
}

// renderTrafficDetail prints all known traffic counters for one user
func (tb *Bot) renderTrafficDetail(ctx context.Context, chatID int64, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	tr, err := tb.svc.LoadTraffic(username)
	if err != nil || tr == nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_no_traffic_for_user"), nil)
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 <b>%s</b>\n", EscapeHTML(username)))
	sb.WriteString(fmt.Sprintf("🌐 %s: <code>%s</code>\n", t("user_ip"), EscapeHTML(tr.IP)))
	sb.WriteString(fmt.Sprintf("📤 %s: <code>%s</code>\n", t("bytes_sent"), FormatBytes(tr.TotalBytesSent)))
	sb.WriteString(fmt.Sprintf("📥 %s: <code>%s</code>\n", t("bytes_received"), FormatBytes(tr.TotalBytesReceived)))
	sb.WriteString(fmt.Sprintf("📦 %s: <code>%s</code>\n", t("total_bytes"), FormatBytes(tr.TotalBytes)))
	sb.WriteString(fmt.Sprintf("⏱️ %s: <code>%s</code>\n", t("last_timestamp"), EscapeHTML(tr.LastTimestamp)))
	markup := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(
				btn("🔙 "+t("tg_btn_back"), CBTraffic+":list:0"),
				btn("🏠 "+t("tg_btn_main_menu"), CBMenu),
			),
		},
	}
	tb.sendText(ctx, chatID, sb.String(), markup)
}
