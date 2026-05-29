/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : users.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:55:00
 * Description  : Telegram bot flows for user management: list,
 *                paginated browse, view, delete, add (auto and
 *                interactive) and per-field edit.
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

// Default values used when creating a user with /adduser
var defaultUserTemplate = User{
	Role:              "user",
	BlockedDomains:    []string{},
	BlockedIPs:        []string{},
	Log:               "yes",
	MaxSessions:       1,
	SessionTTLSeconds: 60,
	MaxSpeedKbps:      2048,
	MaxTotalMB:        1024,
}

// handleUsersCmd opens the users list as a fresh message
func (tb *Bot) handleUsersCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.renderUserList(ctx, update.Message.Chat.ID, 0, 0, lang)
}

// handleAddUserAutoCmd creates a default user instantly
func (tb *Bot) handleAddUserAutoCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.runAddUserAuto(ctx, update.Message.Chat.ID, lang)
}

// handleAddUserInteractiveCmd begins the step by step user creation
func (tb *Bot) handleAddUserInteractiveCmd(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := tb.languageOf(update.Message.From.ID)
	tb.startAddUserInteractive(ctx, update.Message.Chat.ID, update.Message.From.ID, lang)
}

// handleUsersCallback routes all users-namespace callbacks
func (tb *Bot) handleUsersCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	cq := update.CallbackQuery
	if cq == nil || cq.Message.Message == nil {
		return
	}
	tb.ackCallback(ctx, cq, "")
	lang := tb.languageOf(cq.From.ID)
	chatID := cq.Message.Message.Chat.ID
	msgID := cq.Message.Message.ID

	parts := strings.SplitN(cq.Data, ":", 4) // u:action[:arg1[:arg2]]
	if len(parts) < 2 {
		return
	}
	action := parts[1]

	switch action {
	case "list":
		page := 0
		if len(parts) >= 3 {
			if p, err := strconv.Atoi(parts[2]); err == nil {
				page = p
			}
		}
		tb.renderUserList(ctx, chatID, msgID, page, lang)
	case "view":
		if len(parts) < 3 {
			return
		}
		tb.renderUserDetail(ctx, chatID, msgID, parts[2], lang)
	case "del":
		if len(parts) < 3 {
			return
		}
		tb.renderDeleteConfirm(ctx, chatID, msgID, parts[2], lang)
	case "del_yes":
		if len(parts) < 3 {
			return
		}
		tb.performDeleteUser(ctx, chatID, msgID, parts[2], lang)
	case "add_auto":
		tb.runAddUserAuto(ctx, chatID, lang)
	case "add_inter":
		tb.startAddUserInteractive(ctx, chatID, cq.From.ID, lang)
	case "submenu":
		tb.renderUsersSubmenu(ctx, chatID, msgID, lang)
	case "edit":
		if len(parts) < 3 {
			return
		}
		tb.renderEditUserMenu(ctx, chatID, msgID, parts[2], lang)
	case "ef":
		// u:ef:<field>:<username>
		if len(parts) < 4 {
			return
		}
		tb.startEditUserField(ctx, chatID, cq.From.ID, parts[2], parts[3], lang)
	case "role_set":
		// u:role_set:<role>  applied during interactive add flow
		if len(parts) < 3 {
			return
		}
		sess := tb.state.Get(cq.From.ID)
		if sess.Flow != FlowAddUserInteractive || sess.Step != stepAskRole {
			// Stale button after the step already moved on
			return
		}
		tb.removeInlineKeyboard(ctx, chatID, msgID)
		tb.continueAddUserInteractive(ctx, chatID, cq.From.ID, parts[2], lang)
	case "log_set":
		// u:log_set:<yes|no>  applied during interactive add flow
		if len(parts) < 3 {
			return
		}
		sess := tb.state.Get(cq.From.ID)
		if sess.Flow != FlowAddUserInteractive || sess.Step != stepAskLog {
			return
		}
		tb.removeInlineKeyboard(ctx, chatID, msgID)
		tb.continueAddUserInteractive(ctx, chatID, cq.From.ID, parts[2], lang)
	case "ef_role_set":
		// u:ef_role_set:<role>:<username>
		if len(parts) < 4 {
			return
		}
		tb.removeInlineKeyboard(ctx, chatID, msgID)
		tb.applyEditUserFieldFromCallback(ctx, chatID, cq.From.ID, parts[3], "role", parts[2], lang)
	case "ef_log_set":
		// u:ef_log_set:<yes|no>:<username>
		if len(parts) < 4 {
			return
		}
		tb.removeInlineKeyboard(ctx, chatID, msgID)
		tb.applyEditUserFieldFromCallback(ctx, chatID, cq.From.ID, parts[3], "log", parts[2], lang)
	}
}

// renderUsersSubmenu shows the users submenu with add buttons
func (tb *Bot) renderUsersSubmenu(ctx context.Context, chatID int64, msgID int, lang string) {
	text := "👥 <b>" + tb.svc.Translate(lang, "tg_user_submenu_title") + "</b>"
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, UsersSubmenuKeyboard(tb.svc, lang))
		return
	}
	tb.sendText(ctx, chatID, text, UsersSubmenuKeyboard(tb.svc, lang))
}

// renderUserList paginates the user list and renders it as inline buttons
func (tb *Bot) renderUserList(ctx context.Context, chatID int64, msgID int, page int, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	users, err := tb.svc.LoadUsers()
	if err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_loading_users"), nil)
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
		rows = append(rows, kb(btn("➖ "+t("tg_users_empty"), CBNoop)))
	}
	for _, u := range users[start:end] {
		label := fmt.Sprintf("👤 %s (%s)", u.Username, u.Role)
		rows = append(rows, kb(btn(label, fmt.Sprintf("%s:view:%s", CBUsers, u.Username))))
	}
	if nav := PaginationRow(tb.svc, lang, CBUsers+":list", page, totalPages); nav != nil {
		rows = append(rows, nav)
	}
	rows = append(rows, kb(
		btn("➕ "+t("tg_user_add_auto"), CBUsers+":add_auto"),
		btn("➕ "+t("tg_user_add_interactive"), CBUsers+":add_inter"),
	))
	rows = append(rows, kb(btn("🏠 "+t("tg_btn_main_menu"), CBMenu)))

	text := fmt.Sprintf("👥 <b>%s</b>\n%s %d / %d", t("tg_users_title"), t("tg_page"), page+1, totalPages)
	markup := &models.InlineKeyboardMarkup{InlineKeyboard: rows}
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, markup)
		return
	}
	tb.sendText(ctx, chatID, text, markup)
}

// renderUserDetail prints all the user's settings with edit/delete actions
func (tb *Bot) renderUserDetail(ctx context.Context, chatID int64, msgID int, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	u, err := tb.svc.GetUser(username)
	if err != nil || u == nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_user_not_found"), nil)
		return
	}
	text := FormatUserSummary(tb.svc, lang, u)
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, UserDetailKeyboard(tb.svc, lang, username))
		return
	}
	tb.sendText(ctx, chatID, text, UserDetailKeyboard(tb.svc, lang, username))
}

// renderDeleteConfirm prompts the operator before deleting a user
func (tb *Bot) renderDeleteConfirm(ctx context.Context, chatID int64, msgID int, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	text := fmt.Sprintf("⚠️ %s\n<b>%s</b>", t("tg_confirm_delete_user"), EscapeHTML(username))
	yesData := fmt.Sprintf("%s:del_yes:%s", CBUsers, username)
	noData := fmt.Sprintf("%s:view:%s", CBUsers, username)
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, ConfirmKeyboard(tb.svc, lang, yesData, noData))
		return
	}
	tb.sendText(ctx, chatID, text, ConfirmKeyboard(tb.svc, lang, yesData, noData))
}

// performDeleteUser actually removes the user and reports the result
func (tb *Bot) performDeleteUser(ctx context.Context, chatID int64, msgID int, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	if err := tb.svc.DeleteUser(username); err != nil {
		tb.editText(ctx, chatID, msgID, "❌ "+t("tg_err_deleting_user")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	tb.sendText(ctx, chatID, fmt.Sprintf("✅ %s <b>%s</b>", t("tg_user_deleted"), EscapeHTML(username)), nil)
	tb.renderUserList(ctx, chatID, msgID, 0, lang)
}

// runAddUserAuto builds a user with safe defaults
func (tb *Bot) runAddUserAuto(ctx context.Context, chatID int64, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	u := defaultUserTemplate
	u.Username = GenerateUsername()
	u.Password = GeneratePassword()
	// Try a few times to avoid collisions
	for i := 0; i < 5; i++ {
		if existing, err := tb.svc.GetUser(u.Username); err == nil && existing != nil {
			u.Username = GenerateUsername()
			continue
		}
		break
	}
	if err := tb.svc.AddUser(u); err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_creating_user")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	text := "✅ <b>" + t("tg_user_created_auto") + "</b>\n\n" + FormatUserSummary(tb.svc, lang, &u)
	tb.sendText(ctx, chatID, text, nil)
	tb.renderMenu(ctx, chatID, 0, lang)
}

// addUserSteps drive the interactive add flow
const (
	stepAskUsername = iota
	stepAskPassword
	stepAskRole
	stepAskBlockedDomains
	stepAskBlockedIPs
	stepAskLog
	stepAskMaxSessions
	stepAskSessionTTL
	stepAskMaxSpeed
	stepAskMaxTotal
)

// startAddUserInteractive begins the interactive add flow
func (tb *Bot) startAddUserInteractive(ctx context.Context, chatID, userID int64, lang string) {
	tb.state.StartFlow(userID, FlowAddUserInteractive)
	draft := User{
		BlockedDomains: []string{},
		BlockedIPs:     []string{},
		Role:           "user",
		Log:            "no",
		MaxSessions:    1,
	}
	tb.state.SetTemp(userID, TempKeyDraftUser, &draft)
	tb.askAddUserStep(ctx, chatID, userID, lang)
}

// continueAddUserInteractive processes input for the current step
func (tb *Bot) continueAddUserInteractive(ctx context.Context, chatID, userID int64, input string, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	sess := tb.state.Get(userID)
	draftAny, ok := tb.state.GetTemp(userID, TempKeyDraftUser)
	if !ok {
		tb.state.Reset(userID)
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_flow_lost"), nil)
		tb.renderMenu(ctx, chatID, 0, lang)
		return
	}
	draft, ok := draftAny.(*User)
	if !ok || draft == nil {
		tb.state.Reset(userID)
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_flow_lost"), nil)
		tb.renderMenu(ctx, chatID, 0, lang)
		return
	}

	input = strings.TrimSpace(input)
	switch sess.Step {
	case stepAskUsername:
		if strings.EqualFold(input, "auto") || input == "" {
			draft.Username = GenerateUsername()
		} else if err := validateUsername(input); err != nil {
			tb.sendText(ctx, chatID, "❌ "+err.Error(), nil)
			return
		} else {
			if existing, err := tb.svc.GetUser(input); err == nil && existing != nil {
				tb.sendText(ctx, chatID, "❌ "+t("tg_err_username_exists"), nil)
				return
			}
			draft.Username = input
		}
	case stepAskPassword:
		if strings.EqualFold(input, "auto") || input == "" {
			draft.Password = GeneratePassword()
		} else {
			draft.Password = input
		}
	case stepAskRole:
		role := strings.ToLower(input)
		if role != "user" && role != "admin" {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_invalid_role"), nil)
			return
		}
		draft.Role = role
	case stepAskBlockedDomains:
		draft.BlockedDomains = SplitCSV(input)
	case stepAskBlockedIPs:
		draft.BlockedIPs = SplitCSV(input)
	case stepAskLog:
		v := strings.ToLower(input)
		if v != "yes" && v != "no" {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_invalid_yes_no"), nil)
			return
		}
		draft.Log = v
	case stepAskMaxSessions:
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_positive_int"), nil)
			return
		}
		draft.MaxSessions = n
	case stepAskSessionTTL:
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_positive_int"), nil)
			return
		}
		draft.SessionTTLSeconds = n
	case stepAskMaxSpeed:
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_positive_int"), nil)
			return
		}
		draft.MaxSpeedKbps = n
	case stepAskMaxTotal:
		n, err := strconv.Atoi(input)
		if err != nil || n < 0 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_non_negative_int"), nil)
			return
		}
		draft.MaxTotalMB = n
	}

	tb.state.AdvanceFlow(userID)
	sess = tb.state.Get(userID)
	if sess.Step > stepAskMaxTotal {
		if err := tb.svc.AddUser(*draft); err != nil {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_creating_user")+": "+EscapeHTML(err.Error()), nil)
		} else {
			text := "✅ <b>" + t("tg_user_created") + "</b>\n\n" + FormatUserSummary(tb.svc, lang, draft)
			tb.sendText(ctx, chatID, text, nil)
		}
		tb.state.Reset(userID)
		tb.renderMenu(ctx, chatID, 0, lang)
		return
	}
	tb.askAddUserStep(ctx, chatID, userID, lang)
}

// askAddUserStep prompts the operator for the next input
func (tb *Bot) askAddUserStep(ctx context.Context, chatID, userID int64, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	sess := tb.state.Get(userID)
	switch sess.Step {
	case stepAskUsername:
		tb.sendText(ctx, chatID, "📝 "+t("tg_ask_username"), nil)
	case stepAskPassword:
		tb.sendText(ctx, chatID, "🔑 "+t("tg_ask_password"), nil)
	case stepAskRole:
		kb := RolePickKeyboard(tb.svc, lang, CBUsers+":role_set:user", CBUsers+":role_set:admin")
		tb.sendText(ctx, chatID, "🎭 "+t("tg_ask_role"), kb)
	case stepAskBlockedDomains:
		tb.sendText(ctx, chatID, "🌐 "+t("tg_ask_blocked_domains"), nil)
	case stepAskBlockedIPs:
		tb.sendText(ctx, chatID, "🚫 "+t("tg_ask_blocked_ips"), nil)
	case stepAskLog:
		kb := YesNoKeyboard(tb.svc, lang, CBUsers+":log_set:yes", CBUsers+":log_set:no")
		tb.sendText(ctx, chatID, "📝 "+t("tg_ask_log"), kb)
	case stepAskMaxSessions:
		tb.sendText(ctx, chatID, "👥 "+t("tg_ask_max_sessions"), nil)
	case stepAskSessionTTL:
		tb.sendText(ctx, chatID, "⏳ "+t("tg_ask_session_ttl"), nil)
	case stepAskMaxSpeed:
		tb.sendText(ctx, chatID, "⚡ "+t("tg_ask_max_speed"), nil)
	case stepAskMaxTotal:
		tb.sendText(ctx, chatID, "📦 "+t("tg_ask_max_total"), nil)
	}
}

// renderEditUserMenu shows the per-field edit buttons for a user
func (tb *Bot) renderEditUserMenu(ctx context.Context, chatID int64, msgID int, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	text := fmt.Sprintf("✏️ %s: <b>%s</b>\n%s", t("tg_edit_user_title"), EscapeHTML(username), t("tg_edit_user_hint"))
	if msgID > 0 {
		tb.editText(ctx, chatID, msgID, text, EditUserFieldsKeyboard(tb.svc, lang, username))
		return
	}
	tb.sendText(ctx, chatID, text, EditUserFieldsKeyboard(tb.svc, lang, username))
}

// startEditUserField asks the operator for the new value of a single field
func (tb *Bot) startEditUserField(ctx context.Context, chatID, userID int64, field, username, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	if _, err := tb.svc.GetUser(username); err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_user_not_found"), nil)
		return
	}

	// Role and log have inline keyboards instead of text input
	switch field {
	case "role":
		kb := RolePickKeyboard(tb.svc, lang,
			fmt.Sprintf("%s:ef_role_set:user:%s", CBUsers, username),
			fmt.Sprintf("%s:ef_role_set:admin:%s", CBUsers, username))
		tb.sendText(ctx, chatID, "🎭 "+t("tg_ask_role"), kb)
		return
	case "log":
		kb := YesNoKeyboard(tb.svc, lang,
			fmt.Sprintf("%s:ef_log_set:yes:%s", CBUsers, username),
			fmt.Sprintf("%s:ef_log_set:no:%s", CBUsers, username))
		tb.sendText(ctx, chatID, "📝 "+t("tg_ask_log"), kb)
		return
	}

	tb.state.StartFlow(userID, FlowEditUserField)
	tb.state.SetTemp(userID, TempKeyUsername, username)
	tb.state.SetTemp(userID, TempKeyField, field)

	switch field {
	case "password":
		tb.sendText(ctx, chatID, "🔑 "+t("tg_ask_password"), nil)
	case "bdom":
		tb.sendText(ctx, chatID, "🌐 "+t("tg_ask_blocked_domains"), nil)
	case "bips":
		tb.sendText(ctx, chatID, "🚫 "+t("tg_ask_blocked_ips"), nil)
	case "msess":
		tb.sendText(ctx, chatID, "👥 "+t("tg_ask_max_sessions"), nil)
	case "ttl":
		tb.sendText(ctx, chatID, "⏳ "+t("tg_ask_session_ttl"), nil)
	case "speed":
		tb.sendText(ctx, chatID, "⚡ "+t("tg_ask_max_speed"), nil)
	case "total":
		tb.sendText(ctx, chatID, "📦 "+t("tg_ask_max_total"), nil)
	default:
		tb.state.Reset(userID)
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_unknown_field"), nil)
	}
}

// applyEditUserField processes the textual value supplied for an edit
func (tb *Bot) applyEditUserField(ctx context.Context, chatID, userID int64, input string, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	usernameAny, _ := tb.state.GetTemp(userID, TempKeyUsername)
	fieldAny, _ := tb.state.GetTemp(userID, TempKeyField)
	username, _ := usernameAny.(string)
	field, _ := fieldAny.(string)
	if username == "" || field == "" {
		tb.state.Reset(userID)
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_flow_lost"), nil)
		tb.renderMenu(ctx, chatID, 0, lang)
		return
	}
	tb.applyEditUserFieldFromCallback(ctx, chatID, userID, username, field, strings.TrimSpace(input), lang)
}

// applyEditUserFieldFromCallback writes a single field on a user
func (tb *Bot) applyEditUserFieldFromCallback(ctx context.Context, chatID, userID int64, username, field, raw, lang string) {
	t := func(k string) string { return tb.svc.Translate(lang, k) }
	u, err := tb.svc.GetUser(username)
	if err != nil || u == nil {
		tb.state.Reset(userID)
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_user_not_found"), nil)
		return
	}

	switch field {
	case "password":
		if raw == "" {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_empty_value"), nil)
			return
		}
		u.Password = raw
	case "role":
		raw = strings.ToLower(raw)
		if raw != "user" && raw != "admin" {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_invalid_role"), nil)
			return
		}
		u.Role = raw
	case "bdom":
		u.BlockedDomains = SplitCSV(raw)
	case "bips":
		u.BlockedIPs = SplitCSV(raw)
	case "log":
		raw = strings.ToLower(raw)
		if raw != "yes" && raw != "no" {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_invalid_yes_no"), nil)
			return
		}
		u.Log = raw
	case "msess":
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_positive_int"), nil)
			return
		}
		u.MaxSessions = n
	case "ttl":
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_positive_int"), nil)
			return
		}
		u.SessionTTLSeconds = n
	case "speed":
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_positive_int"), nil)
			return
		}
		u.MaxSpeedKbps = n
	case "total":
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			tb.sendText(ctx, chatID, "❌ "+t("tg_err_non_negative_int"), nil)
			return
		}
		u.MaxTotalMB = n
	default:
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_unknown_field"), nil)
		return
	}

	if err := tb.svc.UpdateUser(*u); err != nil {
		tb.sendText(ctx, chatID, "❌ "+t("tg_err_saving_user")+": "+EscapeHTML(err.Error()), nil)
		return
	}
	tb.state.Reset(userID)
	tb.sendText(ctx, chatID, "✅ "+t("tg_user_updated"), nil)
	tb.renderUserDetail(ctx, chatID, 0, username, lang)
}

// validateUsername enforces minimal correctness on a username input
func validateUsername(name string) error {
	if name == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n\r:/,;'\"") {
		return fmt.Errorf("username contains illegal whitespace or punctuation")
	}
	if len(name) > 64 {
		return fmt.Errorf("username is too long")
	}
	return nil
}
