/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : blocked_ips.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 23:00:00
 * Description  : Telegram bot flows for managing the globally blocked
 *                IPs list (list, add, remove).
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

// handleBlockedIPsCmd opens the blocked IPs view from a command
func (tb *Bot) handleBlockedIPsCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderBlockedIPs(ctx, update.Message.Chat.ID, 0, 0, lang)
}

// handleBlockedIPsCallback dispatches the blocked IPs callbacks
func (tb *Bot) handleBlockedIPsCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID
	parts := strings.SplitN(cq.Data, ":", 3) // b:action[:arg]
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
		tb.renderBlockedIPs(ctx, chatID, msgID, page, lang)
	case "del":
		if len(parts) < 3 {
			return
		}
		tb.performRemoveBlockedIP(ctx, chatID, msgID, parts[2], lang)
	case "add":
		tb.startAddBlockedIP(ctx, chatID, cq.From.ID, lang)
	}
}

// renderBlockedIPs prints the blocked IPs with delete buttons
func (tb *Bot) renderBlockedIPs(ctx context.Context, chatID int64, msgID int, page int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	ips, err := tb.svc.LoadBlockedIPs()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_blocked_ips")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	sort.Strings(ips)
	totalPages := (len(ips) + PageSize - 1) / PageSize
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
	if end > len(ips) {
		end = len(ips)
	}

	var rows [][]models.InlineKeyboardButton
	if len(ips) == 0 {
		rows = append(rows, kb(btn("➖ "+t("tg_blocked_ips_empty"), CBNoop)))
	}
	for _, ip := range ips[start:end] {
		rows = append(rows, kb(
			btn("🚫 "+ip, CBNoop),
			btn("🗑️", fmt.Sprintf("%s:del:%s", CBBlockedIPs, ip)),
		))
	}
	if nav := PaginationRow(tb.svc, lang, CBBlockedIPs+":list", page, totalPages); nav != nil {
		rows = append(rows, nav)
	}
	rows = append(rows, kb(btn("➕ "+t("tg_btn_add_ip"), CBBlockedIPs+":add")))
	rows = append(rows, kb(btn("🏠 "+t("tg_btn_main_menu"), CBMenu)))

	text := fmt.Sprintf("🚫 <b>%s</b>\n%s %d / %d", t("blocked_ips"), t("tg_page"), page+1, totalPages)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, markup)
		return
	}
	tb.sendText(ctx, chatID, text, markup)
}

// startAddBlockedIP enters the add IP interactive flow
func (tb *Bot) startAddBlockedIP(ctx context.Context, chatID, userID int64, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	tb.state.StartFlow(userID, FlowAddBlockedIP)
	tb.sendText(ctx, chatID, "➕ "+t("tg_ask_ip"), nil)
}

// applyAddBlockedIP saves the IP entered by the operator
func (tb *Bot) applyAddBlockedIP(ctx context.Context, chatID, userID int64, input string, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	ip := strings.TrimSpace(input)
	if ip == "" {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_empty_value"), nil)
		return
	}
	ips, err := tb.svc.LoadBlockedIPs()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_blocked_ips")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	for _, existing := range ips {
		if existing == ip {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_ip_exists"), nil)
			return
		}
	}
	ips = append(ips, ip)
	if err := tb.svc.SaveBlockedIPs(ips); err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_saving_blocked_ips")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	tb.state.Reset(userID)
	tb.sendText(ctx, chatID, "✅ "+t("tg_ip_added"), nil)
	tb.renderBlockedIPs(ctx, chatID, 0, 0, lang)
}

// performRemoveBlockedIP removes a single IP from the list
func (tb *Bot) performRemoveBlockedIP(ctx context.Context, chatID int64, msgID int, ip, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	ips, err := tb.svc.LoadBlockedIPs()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_blocked_ips")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	updated := make([]string, 0, len(ips))
	found := false
	for _, x := range ips {
		if x == ip {
			found = true
			continue
		}
		updated = append(updated, x)
	}
	if !found {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_ip_not_found"), nil)
		return
	}
	if err := tb.svc.SaveBlockedIPs(updated); err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_saving_blocked_ips")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	tb.sendText(ctx, chatID, "✅ "+t("tg_ip_removed"), nil)
	tb.renderBlockedIPs(ctx, chatID, msgID, 0, lang)
}
