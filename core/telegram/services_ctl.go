/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : services_ctl.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 23:09:00
 * Description  : Telegram bot flows that restart the server or panel
 *                services with a confirmation step.
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
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// handleRestartServerCmd asks for a restart confirmation
func (tb *Bot) handleRestartServerCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderRestartConfirm(ctx, update.Message.Chat.ID, 0, "server", lang)
}

// handleRestartPanelCmd asks for a restart confirmation
func (tb *Bot) handleRestartPanelCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderRestartConfirm(ctx, update.Message.Chat.ID, 0, "panel", lang)
}

// handleServicesCallback dispatches the services namespace
func (tb *Bot) handleServicesCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID
	parts := strings.SplitN(cq.Data, ":", 2) // sv:target  OR  sv:target_yes
	if len(parts) < 2 {
		return
	}
	action := parts[1]

	switch action {
	case "server":
		tb.renderRestartConfirm(ctx, chatID, msgID, "server", lang)
	case "panel":
		tb.renderRestartConfirm(ctx, chatID, msgID, "panel", lang)
	case "server_yes":
		tb.performRestart(ctx, chatID, msgID, "server", lang)
	case "panel_yes":
		tb.performRestart(ctx, chatID, msgID, "panel", lang)
	}
}

// renderRestartConfirm prompts the user to confirm the restart
func (tb *Bot) renderRestartConfirm(ctx context.Context, chatID int64, msgID int, target, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	var prompt string
	var yesData string
	if target == "server" {
		prompt = "⚠️ " + t("tg_confirm_restart_server")
		yesData = CBServices + ":server_yes"
	} else {
		prompt = "⚠️ " + t("tg_confirm_restart_panel")
		yesData = CBServices + ":panel_yes"
	}
	noData := CBMenu
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, prompt, ConfirmKeyboard(tb.svc, lang, yesData, noData))
		return
	}
	tb.sendText(ctx, chatID, prompt, ConfirmKeyboard(tb.svc, lang, yesData, noData))
}

// performRestart runs the actual restart operation
func (tb *Bot) performRestart(ctx context.Context, chatID int64, msgID int, target, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	if target == "server" {
		tb.sendText(ctx, chatID, "⏳ "+t("tg_restarting_server"), nil)
		if err := tb.svc.RestartServerService(); err != nil {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_restart_server")+": "+EscapeHTML(err.Error()), nil)
			return
		}
		tb.sendText(ctx, chatID, "✅ "+t("tg_server_restarted"), nil)
		tb.renderMenu(ctx, chatID, 0, lang)
		return
	}
	// Panel restart is fire-and-forget because the current process will die
	tb.sendText(ctx, chatID, "⏳ "+t("tg_restarting_panel"), nil)
	go func() {
		if err := tb.svc.RestartPanelService(); err != nil {
			tb.svc.Error("Telegram requested panel restart failed", err)
		}
	}()
}
