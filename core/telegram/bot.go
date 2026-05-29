/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : bot.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:50:00
 * Description  : Telegram bot lifecycle (init, start, stop) and core
 *                routing setup. Uses Long Polling via go-telegram/bot.
 *                Bot is decoupled from the panel via PanelService.
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
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Tuning constants for the underlying go-telegram/bot runtime.
// They are intentionally generous so the bot stays responsive even when the
// host panel is busy serving HTTP traffic or doing slow disk I/O.
const (
	// botLongPollTimeout is how long the getUpdates HTTP request is kept
	// open by Telegram. The library cancels the request slightly before
	// this hits, so keeping it at one minute is the upstream-recommended
	// behaviour for Long Polling.
	botLongPollTimeout = 60 * time.Second

	// botShutdownTimeout caps how long Stop() waits for in-flight handlers
	// to finish. After this, the call returns even if some handlers are
	// still running, so the panel can restart cleanly.
	botShutdownTimeout = 10 * time.Second

	// botMinWorkers is the lower bound on the dispatcher worker count.
	// It guarantees parallel update consumption even on tiny machines.
	botMinWorkers = 8
)

// Bot is the high level Telegram bot controller used by the panel
type Bot struct {
	cfg            BotConfig
	svc            PanelService
	state          *StateStore
	api            *bot.Bot
	cancel         context.CancelFunc
	wg             sync.WaitGroup // tracks the polling loop
	handlersWG     sync.WaitGroup // tracks in-flight handler goroutines
	sessionTokenMu sync.RWMutex
	sessionTokens  map[string]string
	mu             sync.Mutex
	running        bool
}

// newBotHTTPClient builds an HTTP client tuned for high-concurrency Telegram
// API usage. The default net/http transport caps idle connections per host at
// 2, which serialises bursts of SendMessage / EditMessage calls; we lift that
// limit so dozens of handlers can talk to api.telegram.org in parallel.
func newBotHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       0, // unlimited
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		// Use the long-polling timeout as ceiling for getUpdates. Regular API
		// calls (sendMessage, etc.) finish well below this, so the same client
		// can safely serve both.
		Timeout: botLongPollTimeout + 10*time.Second,
	}
}

// botWorkerCount picks a sensible number of dispatcher workers based on the
// host CPU count. More workers means more updates can be pulled off the
// internal channel and dispatched simultaneously.
func botWorkerCount() int {
	n := runtime.NumCPU() * 2
	if n < botMinWorkers {
		n = botMinWorkers
	}
	return n
}

// New creates a Bot wrapper. It does not contact Telegram yet.
func New(cfg BotConfig, svc PanelService) *Bot {
	return &Bot{
		cfg:           cfg,
		svc:           svc,
		state:         NewStateStore(StateFilePath),
		sessionTokens: make(map[string]string),
	}
}

// IsRunning reports whether the bot polling loop is active
func (tb *Bot) IsRunning() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.running
}

// Start initializes the bot and begins Long Polling in a background goroutine.
// Returns nil and logs the reason when the bot is disabled or misconfigured so
// the panel keeps running normally.
func (tb *Bot) Start(parentCtx context.Context) error {
	tb.mu.Lock()
	if tb.running {
		tb.mu.Unlock()
		return nil
	}
	if !tb.cfg.Enabled {
		tb.mu.Unlock()
		tb.svc.Info("Telegram bot disabled in panel config; skipping startup")
		return nil
	}
	if strings.TrimSpace(tb.cfg.Token) == "" {
		tb.mu.Unlock()
		tb.svc.Warning("Telegram bot enabled but API token is empty; skipping startup")
		return nil
	}
	if len(tb.cfg.Admins) == 0 {
		tb.svc.Warning("Telegram bot has no admin IDs configured; only admins are allowed to interact, no one will be able to use the bot")
	}

	if err := tb.state.Load(); err != nil {
		tb.svc.Warning(fmt.Sprintf("Failed to load Telegram state file: %v", err))
	}

	// Custom HTTP client with a large connection pool keeps every Telegram
	// API call non-blocking, even when dozens of handlers run concurrently.
	httpClient := newBotHTTPClient()
	workers := botWorkerCount()

	opts := []bot.Option{
		bot.WithDefaultHandler(tb.defaultHandler),
		// trackerMiddleware MUST be the outermost middleware so every spawned
		// handler goroutine is registered in the WaitGroup before any other
		// logic runs.
		bot.WithMiddlewares(tb.trackerMiddleware, tb.adminOnlyMiddleware, tb.recoverMiddleware),
		bot.WithHTTPClient(botLongPollTimeout, httpClient),
		bot.WithWorkers(workers),
	}

	api, err := bot.New(tb.cfg.Token, opts...)
	if err != nil {
		tb.mu.Unlock()
		return fmt.Errorf("failed to initialize Telegram bot: %w", err)
	}

	tb.registerHandlers(api)

	ctx, cancel := context.WithCancel(parentCtx)
	tb.api = api
	tb.cancel = cancel
	tb.running = true
	tb.wg.Add(1)
	tb.mu.Unlock()

	// publishCommands does a blocking HTTP call to Telegram; run it in the
	// background so Start() never delays the panel boot sequence.
	go tb.publishCommands(ctx)

	go func() {
		defer tb.wg.Done()
		tb.svc.Info(fmt.Sprintf("Telegram bot started (Long Polling, %d workers). Listening for updates...", workers))
		api.Start(ctx)
		tb.mu.Lock()
		tb.running = false
		tb.mu.Unlock()
		tb.svc.Info("Telegram bot polling loop terminated")
	}()

	return nil
}

// Stop cancels the polling loop and waits for it to finish. It also waits a
// bounded amount of time for in-flight handlers to drain so the panel can
// restart cleanly without leaking goroutines.
func (tb *Bot) Stop() {
	tb.mu.Lock()
	if !tb.running {
		tb.mu.Unlock()
		return
	}
	cancel := tb.cancel
	tb.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	tb.wg.Wait()

	// Wait for handler goroutines spawned by the library's async dispatch,
	// but never block the shutdown sequence indefinitely.
	done := make(chan struct{})
	go func() {
		tb.handlersWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(botShutdownTimeout):
		tb.svc.Warning("Telegram bot stop timed out waiting for in-flight handlers to finish")
	}
}

// registerHandlers wires command and callback handlers into the underlying bot
func (tb *Bot) registerHandlers(api *bot.Bot) {
	// Commands (MatchTypeCommand expects the pattern without a leading slash)
	api.RegisterHandler(bot.HandlerTypeMessageText, "start", bot.MatchTypeCommand, tb.handleStart)
	api.RegisterHandler(bot.HandlerTypeMessageText, "language", bot.MatchTypeCommand, tb.handleLanguageCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "menu", bot.MatchTypeCommand, tb.handleMenuCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "help", bot.MatchTypeCommand, tb.handleHelpCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "cancel", bot.MatchTypeCommand, tb.handleCancelCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "users", bot.MatchTypeCommand, tb.handleUsersCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "adduser", bot.MatchTypeCommand, tb.handleAddUserAutoCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "adduser_interactive", bot.MatchTypeCommand, tb.handleAddUserInteractiveCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "server", bot.MatchTypeCommand, tb.handleServerCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "blockedips", bot.MatchTypeCommand, tb.handleBlockedIPsCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "logs", bot.MatchTypeCommand, tb.handleLogsCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "blockedaccess", bot.MatchTypeCommand, tb.handleBlockedAccessCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "traffic", bot.MatchTypeCommand, tb.handleTrafficCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "sessions", bot.MatchTypeCommand, tb.handleSessionsCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "restart_server", bot.MatchTypeCommand, tb.handleRestartServerCmd)
	api.RegisterHandler(bot.HandlerTypeMessageText, "restart_panel", bot.MatchTypeCommand, tb.handleRestartPanelCmd)

	// Callback query namespaces - dispatched to per-domain handlers
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBNoop, bot.MatchTypeExact, tb.handleNoopCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBMenu, bot.MatchTypeExact, tb.handleMenuCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBHelp, bot.MatchTypeExact, tb.handleHelpCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBCancel, bot.MatchTypeExact, tb.handleCancelCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBLang+":", bot.MatchTypePrefix, tb.handleLanguageCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBUsers+":", bot.MatchTypePrefix, tb.handleUsersCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBServer+":", bot.MatchTypePrefix, tb.handleServerCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBBlockedIPs+":", bot.MatchTypePrefix, tb.handleBlockedIPsCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBLogs+":", bot.MatchTypePrefix, tb.handleLogsCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBBlocked+":", bot.MatchTypePrefix, tb.handleBlockedAccessCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBTraffic+":", bot.MatchTypePrefix, tb.handleTrafficCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBSessions+":", bot.MatchTypePrefix, tb.handleSessionsCallback)
	api.RegisterHandler(bot.HandlerTypeCallbackQueryData, CBServices+":", bot.MatchTypePrefix, tb.handleServicesCallback)
}

// publishCommands sets the visible command list for the bot in Telegram
func (tb *Bot) publishCommands(ctx context.Context) {
	commands := []models.BotCommand{
		{Command: "start", Description: "Start the bot and pick language"},
		{Command: "menu", Description: "Open the main menu"},
		{Command: "language", Description: "Change interface language"},
		{Command: "users", Description: "Open the users menu"},
		{Command: "adduser", Description: "Create a user with default values"},
		{Command: "adduser_interactive", Description: "Create a user step by step"},
		{Command: "server", Description: "View or edit server configuration"},
		{Command: "blockedips", Description: "Manage globally blocked IPs"},
		{Command: "logs", Description: "Browse user access logs"},
		{Command: "blockedaccess", Description: "Browse blocked access logs"},
		{Command: "traffic", Description: "Show per user traffic usage"},
		{Command: "sessions", Description: "List and revoke sessions"},
		{Command: "restart_server", Description: "Restart the server service"},
		{Command: "restart_panel", Description: "Restart the panel service"},
		{Command: "cancel", Description: "Cancel the current interactive flow"},
		{Command: "help", Description: "Show help"},
	}
	if _, err := tb.api.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: commands}); err != nil {
		tb.svc.Warning(fmt.Sprintf("Failed to publish Telegram bot commands: %v", err))
	}
}

// trackerMiddleware registers each handler invocation in the bot WaitGroup so
// Stop() can wait for in-flight work to finish. The go-telegram/bot library
// already dispatches every handler in its own goroutine (see ProcessUpdate),
// which means slow handlers never block one another; this middleware just
// gives us visibility into that pool for clean shutdown.
func (tb *Bot) trackerMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		tb.handlersWG.Add(1)
		defer tb.handlersWG.Done()
		next(ctx, b, update)
	}
}

// recoverMiddleware shields the polling loop from handler panics
func (tb *Bot) recoverMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		defer func() {
			if r := recover(); r != nil {
				tb.svc.Error("Telegram handler panic", fmt.Errorf("%v", r))
			}
		}()
		next(ctx, b, update)
	}
}

// adminOnlyMiddleware drops any update originating from a non-admin user.
// It guarantees the bot never reacts to unauthorized callers.
func (tb *Bot) adminOnlyMiddleware(next bot.HandlerFunc) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		userID := UserIDFromUpdate(update)
		if userID == 0 {
			return
		}
		if !tb.isAdmin(userID) {
			tb.svc.Warning(fmt.Sprintf("Telegram bot ignored update from unauthorized user ID %d", userID))
			// Reply once so the user understands their request was dropped
			chatID := ChatIDFromUpdate(update)
			if chatID != 0 {
				_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: chatID,
					Text:   "⛔ You are not authorized to use this bot.",
				})
			}
			return
		}
		next(ctx, b, update)
	}
}

// isAdmin returns true when the given Telegram user is in the admins list
func (tb *Bot) isAdmin(userID int64) bool {
	for _, id := range tb.cfg.Admins {
		if id == userID {
			return true
		}
	}
	return false
}
