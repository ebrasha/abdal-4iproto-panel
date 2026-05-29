/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : handlers.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:52:00
 * Description  : Top-level Telegram handlers: /start, /language,
 *                /menu, /help, /cancel and the default text handler
 *                that drives interactive multi-step flows.
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
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// languageOf returns the stored language for a user, defaulting to "en"
func (tb *Bot) languageOf(userID int64) string {
	lang := tb.state.Language(userID)
	if lang == "" {
		return "en"
	}
	return lang
}

// sendText is a thin wrapper used for normal HTML messages
func (tb *Bot) sendText(ctx context.Context, chatID int64, text string, kb *models.InlineKeyboardMarkup) {
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	if _, err := tb.api.SendMessage(ctx, params); err != nil {
		tb.svc.Error("Telegram SendMessage failed", err)
	}
}

// editText edits the existing message that hosts a callback button menu
func (tb *Bot) editText(ctx context.Context, chatID int64, messageID int, text string, kb *models.InlineKeyboardMarkup) {
	params := &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	if _, err := tb.api.EditMessageText(ctx, params); err != nil {
		// Falls back to a fresh message if the original is too old or identical
		tb.sendText(ctx, chatID, text, kb)
	}
}

// ackCallback always answers a callback query so the spinner stops in the client
func (tb *Bot) ackCallback(ctx context.Context, cq *models.CallbackQuery, text string) {
	_, _ = tb.api.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: cq.ID,
		Text:            text,
	})
}

// removeInlineKeyboard clears the inline keyboard attached to an old message
// so the operator cannot click the same one-shot button twice.
func (tb *Bot) removeInlineKeyboard(ctx context.Context, chatID int64, messageID int) {
	if messageID <= 0 {
		return
	}
	_, _ = tb.api.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      chatID,
		MessageID:   messageID,
		ReplyMarkup: nil,
	})
}

// renderMenu produces or refreshes the main menu in the chat
func (tb *Bot) renderMenu(ctx context.Context, chatID int64, messageID int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	text := "🏠 <b>" + t("tg_menu_title") + "</b>\n\n" + t("tg_menu_help_hint")
	kb := MainMenuKeyboard(tb.svc, lang)
	if messageID > 0 {
		tb.editText(ctx, chatID, messageID, text, kb)
		return
	}
	tb.sendText(ctx, chatID, text, kb)
}

// renderLanguagePick shows the language picker
func (tb *Bot) renderLanguagePick(ctx context.Context, chatID int64, messageID int) {
	text := "🌐 <b>Please choose a language</b>\n🌐 <b>لطفاً زبان را انتخاب کنید</b>"
	if messageID > 0 {
		tb.editText(ctx, chatID, messageID, text, LanguageKeyboard())
		return
	}
	tb.sendText(ctx, chatID, text, LanguageKeyboard())
}

// handleStart is the entry point for new users
func (tb *Bot) handleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	tb.state.Reset(update.Message.From.ID)
	lang := tb.state.Language(update.Message.From.ID)
	if lang == "" {
		tb.renderLanguagePick(ctx, update.Message.Chat.ID, 0)
		return
	}
	tb.sendText(ctx, update.Message.Chat.ID, tb.svc.Translate(lang, "tg_welcome_back"), nil)
	tb.renderMenu(ctx, update.Message.Chat.ID, 0, lang)
}

// handleLanguageCmd lets the user change the language at any time
func (tb *Bot) handleLanguageCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	tb.renderLanguagePick(ctx, update.Message.Chat.ID, 0)
}

// handleMenuCmd shows the main menu
func (tb *Bot) handleMenuCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	tb.state.Reset(update.Message.From.ID)
	tb.renderMenu(ctx, update.Message.Chat.ID, 0, tb.languageOf(update.Message.From.ID))
}

// handleHelpCmd prints a short list of available actions
func (tb *Bot) handleHelpCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	tb.sendHelp(ctx, update.Message.Chat.ID, tb.languageOf(update.Message.From.ID))
}

func (tb *Bot) sendHelp(ctx context.Context, chatID int64, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	var sb strings.Builder
	sb.WriteString("ℹ️ <b>" + t("tg_help_title") + "</b>\n\n")
	sb.WriteString("/start - " + t("tg_help_start") + "\n")
	sb.WriteString("/menu - " + t("tg_help_menu") + "\n")
	sb.WriteString("/language - " + t("tg_help_language") + "\n")
	sb.WriteString("/users - " + t("tg_help_users") + "\n")
	sb.WriteString("/adduser - " + t("tg_help_adduser") + "\n")
	sb.WriteString("/adduser_interactive - " + t("tg_help_adduser_interactive") + "\n")
	sb.WriteString("/server - " + t("tg_help_server") + "\n")
	sb.WriteString("/blockedips - " + t("tg_help_blockedips") + "\n")
	sb.WriteString("/logs - " + t("tg_help_logs") + "\n")
	sb.WriteString("/blockedaccess - " + t("tg_help_blockedaccess") + "\n")
	sb.WriteString("/traffic - " + t("tg_help_traffic") + "\n")
	sb.WriteString("/sessions - " + t("tg_help_sessions") + "\n")
	sb.WriteString("/restart_server - " + t("tg_help_restart_server") + "\n")
	sb.WriteString("/restart_panel - " + t("tg_help_restart_panel") + "\n")
	sb.WriteString("/cancel - " + t("tg_help_cancel") + "\n")
	tb.sendText(ctx, chatID, sb.String(), nil)
}

// handleCancelCmd aborts an interactive flow
func (tb *Bot) handleCancelCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	tb.state.Reset(update.Message.From.ID)
	lang := tb.languageOf(update.Message.From.ID)
	tb.sendText(ctx, update.Message.Chat.ID, "✅ "+tb.svc.Translate(lang, "tg_flow_cancelled"), nil)
	tb.renderMenu(ctx, update.Message.Chat.ID, 0, lang)
}

// handleNoopCallback simply acknowledges decorative buttons
func (tb *Bot) handleNoopCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		tb.ackCallback(ctx, update.CallbackQuery, "")
	}
}

// handleMenuCallback returns the user to the main menu
func (tb *Bot) handleMenuCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, update.CallbackQuery, "")
	tb.state.Reset(update.CallbackQuery.From.ID)
	lang := tb.languageOf(update.CallbackQuery.From.ID)
	tb.renderMenu(ctx, update.CallbackQuery.Message.Message.Chat.ID, update.CallbackQuery.Message.Message.ID, lang)
}

// handleHelpCallback shows the help block under the menu
func (tb *Bot) handleHelpCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, update.CallbackQuery, "")
	lang := tb.languageOf(update.CallbackQuery.From.ID)
	tb.sendHelp(ctx, update.CallbackQuery.Message.Message.Chat.ID, lang)
}

// handleCancelCallback aborts a flow when pressed from an inline button
func (tb *Bot) handleCancelCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil || update.CallbackQuery.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, update.CallbackQuery, "")
	tb.state.Reset(update.CallbackQuery.From.ID)
	lang := tb.languageOf(update.CallbackQuery.From.ID)
	tb.renderMenu(ctx, update.CallbackQuery.Message.Message.Chat.ID, update.CallbackQuery.Message.Message.ID, lang)
}

// handleLanguageCallback either shows the picker (data == "lang:pick")
// or applies the chosen language (data == "lang:en" / "lang:fa").
func (tb *Bot) handleLanguageCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")

	parts := strings.SplitN(cq.Data, ":", 2)
	if len(parts) != 2 {
		return
	}
	choice := parts[1]
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID

	if choice == "pick" {
		tb.renderLanguagePick(ctx, chatID, msgID)
		return
	}
	if choice != "en" && choice != "fa" {
		return
	}
	if err := tb.state.SetLanguage(cq.From.ID, choice); err != nil {
		tb.svc.Warning(fmt.Sprintf("Failed to persist Telegram language for user %d: %v", cq.From.ID, err))
	}
	greet := tb.svc.Translate(choice, "tg_welcome")
	tb.sendText(ctx, chatID, greet, nil)
	tb.renderMenu(ctx, chatID, 0, choice)
}

// defaultHandler routes free text messages into active flows when present.
// It also gives a helpful hint for messages that have no matching command.
func (tb *Bot) defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}
	userID := update.Message.From.ID
	chatID := update.Message.Chat.ID
	lang := tb.languageOf(userID)

	// Ignore commands the bot does not know to avoid duplicate hints
	if strings.HasPrefix(update.Message.Text, "/") {
		tb.sendText(ctx, chatID, "❓ "+tb.svc.Translate(lang, "tg_unknown_command"), nil)
		return
	}

	sess := tb.state.Get(userID)
	switch sess.Flow {
	case FlowAddUserInteractive:
		tb.continueAddUserInteractive(ctx, chatID, userID, update.Message.Text, lang)
	case FlowEditUserField:
		tb.applyEditUserField(ctx, chatID, userID, update.Message.Text, lang)
	case FlowEditServerField:
		tb.applyEditServerField(ctx, chatID, userID, update.Message.Text, lang)
	case FlowAddBlockedIP:
		tb.applyAddBlockedIP(ctx, chatID, userID, update.Message.Text, lang)
	default:
		tb.sendText(ctx, chatID, "❓ "+tb.svc.Translate(lang, "tg_use_menu_hint"), nil)
		tb.renderMenu(ctx, chatID, 0, lang)
	}
}
