/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : keyboards.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:47:00
 * Description  : Inline keyboard builders for the Telegram bot UI.
 *                All buttons use InlineKeyboardButton with emoji.
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package telegram

import (
	"fmt"

	"github.com/go-telegram/bot/models"
)

// Callback data prefixes. Keep them short to respect 64 byte limits.
const (
	CBNoop        = "noop"
	CBMenu        = "menu"
	CBLang        = "lang"     // lang:<code>
	CBHelp        = "help"
	CBCancel      = "cancel"
	CBUsers       = "u"        // users namespace
	CBServer      = "s"        // server config namespace
	CBBlockedIPs  = "b"        // blocked ips namespace
	CBLogs        = "lg"       // user logs namespace
	CBBlocked     = "ba"       // blocked access namespace
	CBTraffic     = "tr"       // traffic namespace
	CBSessions    = "se"       // sessions namespace
	CBServices    = "sv"       // service control namespace
)

// PageSize controls how many items are shown per page in list views
const PageSize = 8

// kb is a small helper for building one row of buttons
func kb(buttons ...models.InlineKeyboardButton) []models.InlineKeyboardButton {
	return buttons
}

func btn(text, data string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: data,
	}
}

// LanguageKeyboard returns the initial language selection keyboard
func LanguageKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(btn("🇬🇧 English", CBLang+":en"), btn("🇮🇷 فارسی", CBLang+":fa")),
		},
	}
}

// MainMenuKeyboard returns the main menu shown to admins
func MainMenuKeyboard(svc PanelService, lang string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(
				btn("👥 "+t("tg_menu_users"), CBUsers+":list:0"),
				btn("⚙️ "+t("tg_menu_server"), CBServer+":view"),
			),
			kb(
				btn("🚫 "+t("tg_menu_blocked_ips"), CBBlockedIPs+":list:0"),
				btn("📊 "+t("tg_menu_traffic"), CBTraffic+":list:0"),
			),
			kb(
				btn("📋 "+t("tg_menu_logs"), CBLogs+":list:0"),
				btn("🛑 "+t("tg_menu_blocked_access"), CBBlocked+":list:0"),
			),
			kb(
				btn("🔗 "+t("tg_menu_sessions"), CBSessions+":list:0"),
				btn("🔁 "+t("tg_menu_restart_server"), CBServices+":server"),
			),
			kb(
				btn("🔄 "+t("tg_menu_restart_panel"), CBServices+":panel"),
				btn("🌐 "+t("tg_menu_language"), CBLang+":pick"),
			),
			kb(btn("ℹ️ "+t("tg_menu_help"), CBHelp)),
		},
	}
}

// UsersSubmenuKeyboard returns the users-related actions
func UsersSubmenuKeyboard(svc PanelService, lang string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(btn("➕ "+t("tg_user_add_auto"), CBUsers+":add_auto")),
			kb(btn("➕ "+t("tg_user_add_interactive"), CBUsers+":add_inter")),
			kb(btn("🏠 "+t("tg_btn_main_menu"), CBMenu)),
		},
	}
}

// PaginationRow builds a navigation row with prev / next / back buttons
func PaginationRow(svc PanelService, lang, prefix string, page, totalPages int) []models.InlineKeyboardButton {
	t := func(k string) string { return svc.Translate(lang, k) }
	var row []models.InlineKeyboardButton
	if page > 0 {
		row = append(row, btn("⬅️ "+t("tg_btn_prev"), fmt.Sprintf("%s:%d", prefix, page-1)))
	}
	if page < totalPages-1 {
		row = append(row, btn(t("tg_btn_next")+" ➡️", fmt.Sprintf("%s:%d", prefix, page+1)))
	}
	if len(row) == 0 {
		return nil
	}
	return row
}

// ConfirmKeyboard returns a Yes / No keyboard for destructive confirmations
func ConfirmKeyboard(svc PanelService, lang, yesData, noData string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(
				btn("✅ "+t("tg_btn_yes"), yesData),
				btn("❌ "+t("tg_btn_no"), noData),
			),
		},
	}
}

// UserDetailKeyboard returns actions for a single user view
func UserDetailKeyboard(svc PanelService, lang, username string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(
				btn("✏️ "+t("tg_btn_edit"), CBUsers+":edit:"+username),
				btn("🗑️ "+t("tg_btn_delete"), CBUsers+":del:"+username),
			),
			kb(
				btn("🔙 "+t("tg_btn_back"), CBUsers+":list:0"),
				btn("🏠 "+t("tg_btn_main_menu"), CBMenu),
			),
		},
	}
}

// EditUserFieldsKeyboard returns the per-field editors for a user
func EditUserFieldsKeyboard(svc PanelService, lang, username string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	p := func(field string) string { return fmt.Sprintf("%s:ef:%s:%s", CBUsers, field, username) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(btn("🔑 "+t("password"), p("password")), btn("🎭 "+t("role"), p("role"))),
			kb(btn("🌐 "+t("blocked_domains"), p("bdom")), btn("🚫 "+t("blocked_ips"), p("bips"))),
			kb(btn("📝 "+t("log"), p("log")), btn("👥 "+t("max_sessions"), p("msess"))),
			kb(btn("⏳ "+t("session_ttl_seconds"), p("ttl")), btn("⚡ "+t("max_speed_kbps"), p("speed"))),
			kb(btn("📦 "+t("max_total_mb"), p("total"))),
			kb(btn("🔙 "+t("tg_btn_back"), CBUsers+":view:"+username), btn("🏠 "+t("tg_btn_main_menu"), CBMenu)),
		},
	}
}

// EditServerFieldsKeyboard returns the per-field editors for server config
func EditServerFieldsKeyboard(svc PanelService, lang string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	p := func(field string) string { return fmt.Sprintf("%s:ef:%s", CBServer, field) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(btn("🔌 "+t("ports"), p("ports")), btn("🐚 "+t("shell"), p("shell"))),
			kb(btn("🔁 "+t("max_auth_attempts"), p("max_auth")), btn("📛 "+t("server_version"), p("ver"))),
			kb(btn("🔑 "+t("private_key_file"), p("priv")), btn("🗝️ "+t("public_key_file"), p("pub"))),
			kb(btn("🔙 "+t("tg_btn_back"), CBMenu), btn("🏠 "+t("tg_btn_main_menu"), CBMenu)),
		},
	}
}

// YesNoRoleKeyboard returns Yes/No buttons used in interactive add-user flow
func YesNoKeyboard(svc PanelService, lang, yesData, noData string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(
				btn("✅ "+t("yes"), yesData),
				btn("❌ "+t("no"), noData),
			),
		},
	}
}

// RolePickKeyboard returns user role picker
func RolePickKeyboard(svc PanelService, lang, userPrefix, adminPrefix string) *models.InlineKeyboardMarkup {
	t := func(k string) string { return svc.Translate(lang, k) }
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			kb(
				btn("👤 "+t("user"), userPrefix),
				btn("👑 "+t("admin"), adminPrefix),
			),
		},
	}
}
