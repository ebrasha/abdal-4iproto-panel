/*
 **********************************************************************
 * -------------------------------------------------------------------
 * Project Name : Abdal 4iProto Panel
 * File Name    : state.go
 * Author       : Ebrahim Shafiei (EbraSha)
 * Email        : Prof.Shafiei@Gmail.com
 * Created On   : 2026-05-29 22:46:00
 * Description  : In-memory per user session store with on-disk
 *                persistence of language preference. Tracks active
 *                interactive flows used by handlers.
 * -------------------------------------------------------------------
 *
 * "Coding is an engaging and beloved hobby for me. I passionately and insatiably pursue knowledge in cybersecurity and programming."
 * – Ebrahim Shafiei
 *
 **********************************************************************
 */

package telegram

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// StateFilePath is the persistence file for language preferences
const StateFilePath = "data/telegram_users.json"

// Flow identifiers used by interactive handlers
const (
	FlowNone               = ""
	FlowAddUserInteractive = "add_user_interactive"
	FlowEditUserField      = "edit_user_field"
	FlowEditServerField    = "edit_server_field"
	FlowAddBlockedIP       = "add_blocked_ip"
)

// Common temp keys used by flows
const (
	TempKeyUsername  = "username"
	TempKeyField     = "field"
	TempKeyDraftUser = "draft_user"
)

// UserSession captures runtime state for a single Telegram chat
type UserSession struct {
	UserID    int64
	Language  string
	Flow      string
	Step      int
	Temp      map[string]interface{}
	LastMsgID int
}

// StateStore holds all user sessions in memory and persists languages to disk
type StateStore struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
	filePath string
}

type persistedState struct {
	Languages map[string]string `json:"languages"`
}

// NewStateStore returns a StateStore that persists to the given file
func NewStateStore(path string) *StateStore {
	return &StateStore{
		sessions: make(map[int64]*UserSession),
		filePath: path,
	}
}

// Get returns the session for the user, creating one if needed
func (s *StateStore) Get(userID int64) *UserSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[userID]
	if !ok {
		sess = &UserSession{
			UserID: userID,
			Temp:   make(map[string]interface{}),
		}
		s.sessions[userID] = sess
	}
	if sess.Temp == nil {
		sess.Temp = make(map[string]interface{})
	}
	return sess
}

// Language returns the saved language for a user or empty string if none
func (s *StateStore) Language(userID int64) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sess, ok := s.sessions[userID]; ok {
		return sess.Language
	}
	return ""
}

// SetLanguage persists the language preference for a user
func (s *StateStore) SetLanguage(userID int64, lang string) error {
	s.mu.Lock()
	sess, ok := s.sessions[userID]
	if !ok {
		sess = &UserSession{UserID: userID, Temp: make(map[string]interface{})}
		s.sessions[userID] = sess
	}
	sess.Language = lang
	s.mu.Unlock()
	return s.Save()
}

// StartFlow sets an interactive flow on the user session
func (s *StateStore) StartFlow(userID int64, flow string) {
	sess := s.Get(userID)
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Flow = flow
	sess.Step = 0
	sess.Temp = make(map[string]interface{})
}

// AdvanceFlow increments the current flow step
func (s *StateStore) AdvanceFlow(userID int64) {
	sess := s.Get(userID)
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Step++
}

// SetTemp stores a value in the session temp bag
func (s *StateStore) SetTemp(userID int64, key string, value interface{}) {
	sess := s.Get(userID)
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Temp[key] = value
}

// GetTemp reads a value from the session temp bag
func (s *StateStore) GetTemp(userID int64, key string) (interface{}, bool) {
	sess := s.Get(userID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := sess.Temp[key]
	return v, ok
}

// Reset clears the active flow and temp data for a user
func (s *StateStore) Reset(userID int64) {
	sess := s.Get(userID)
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Flow = FlowNone
	sess.Step = 0
	sess.Temp = make(map[string]interface{})
}

// Load reads language preferences from disk into memory
func (s *StateStore) Load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var ps persistedState
	if err := json.Unmarshal(data, &ps); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for idStr, lang := range ps.Languages {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id == 0 {
			continue
		}
		s.sessions[id] = &UserSession{
			UserID:   id,
			Language: lang,
			Temp:     make(map[string]interface{}),
		}
	}
	return nil
}

// Save writes language preferences to disk
func (s *StateStore) Save() error {
	s.mu.RLock()
	ps := persistedState{Languages: make(map[string]string)}
	for id, sess := range s.sessions {
		if sess.Language != "" {
			ps.Languages[strconv.FormatInt(id, 10)] = sess.Language
		}
	}
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}
