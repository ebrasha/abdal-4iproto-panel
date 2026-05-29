/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : server_config.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:58:00
 * Description  : Telegram bot flows for viewing and editing the
 *                Abdal 4iProto Server configuration.
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
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// handleServerCmd renders the server config from a chat command
func (tb *Bot) handleServerCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderServerConfig(ctx, update.Message.Chat.ID, 0, lang)
}

// handleServerCallback routes server-namespace callbacks
func (tb *Bot) handleServerCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID
	parts := strings.SplitN(cq.Data, ":", 3) // s:action[:arg]
	if len(parts) < 2 {
		return
	}

	switch parts[1] {
	case "view":
		tb.renderServerConfig(ctx, chatID, msgID, lang)
	case "ef":
		if len(parts) < 3 {
			return
		}
		tb.startEditServerField(ctx, chatID, cq.From.ID, parts[2], lang)
	}
}

// renderServerConfig shows the current server configuration
func (tb *Bot) renderServerConfig(ctx context.Context, chatID int64, msgID int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	cfg, err := tb.svc.LoadServerConfig()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_server")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	text := FormatServerConfig(tb.svc, lang, cfg)
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, EditServerFieldsKeyboard(tb.svc, lang))
		return
	}
	tb.sendText(ctx, chatID, text, EditServerFieldsKeyboard(tb.svc, lang))
}

// startEditServerField prepares the prompt for the chosen field
func (tb *Bot) startEditServerField(ctx context.Context, chatID, userID int64, field, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	tb.state.StartFlow(userID, FlowEditServerField)
	tb.state.SetTemp(userID, TempKeyField, field)

	switch field {
	case "ports":
		tb.sendText(ctx, chatID, "🔌 "+t("tg_ask_ports"), nil)
	case "shell":
		tb.sendText(ctx, chatID, "🐚 "+t("tg_ask_shell"), nil)
	case "max_auth":
		tb.sendText(ctx, chatID, "🔁 "+t("tg_ask_max_auth"), nil)
	case "ver":
		tb.sendText(ctx, chatID, "📛 "+t("tg_ask_server_version"), nil)
	case "priv":
		tb.sendText(ctx, chatID, "🔑 "+t("tg_ask_private_key"), nil)
	case "pub":
		tb.sendText(ctx, chatID, "🗝️ "+t("tg_ask_public_key"), nil)
	default:
		tb.state.Reset(userID)
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_unknown_field"), nil)
	}
}

// applyEditServerField commits the operator's input to the server config
func (tb *Bot) applyEditServerField(ctx context.Context, chatID, userID int64, input string, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	fieldAny, _ := tb.state.GetTemp(userID, TempKeyField)
	field, _ := fieldAny.(string)
	if field == "" {
		tb.state.Reset(userID)
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_flow_lost"), nil)
		tb.renderMenu(ctx, chatID, 0, lang)
		return
	}
	cfg, err := tb.svc.LoadServerConfig()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_server")+": "+EscapeHTML(err.Error()), nil)
		return
	}

	input = strings.TrimSpace(input)
	switch field {
	case "ports":
		parts := SplitCSV(input)
		if len(parts) == 0 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_ports_empty"), nil)
			return
		}
		ports := make([]int, 0, len(parts))
		for _, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil || n < 1 || n > 65535 {
				tb.sendText(ctx, chatID, fmt.Sprintf("❌ %s: %s", t("tg_err_invalid_port"), EscapeHTML(p)), nil)
				return
			}
			ports = append(ports, n)
		}
		cfg.Ports = ports
	case "shell":
		if input == "" {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_empty_value"), nil)
			return
		}
		cfg.Shell = input
	case "max_auth":
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_positive_int"), nil)
			return
		}
		cfg.MaxAuthAttempts = n
	case "ver":
		if input == "" {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_empty_value"), nil)
			return
		}
		cfg.ServerVersion = input
	case "priv":
		cfg.PrivateKeyFile = input
	case "pub":
		cfg.PublicKeyFile = input
	default:
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_unknown_field"), nil)
		return
	}

	if err := tb.svc.SaveServerConfig(cfg); err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_saving_server")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	tb.state.Reset(userID)
	tb.sendText(ctx, chatID, "✅ "+t("tg_server_updated"), nil)
	tb.renderServerConfig(ctx, chatID, 0, lang)
}
