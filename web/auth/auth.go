// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type User struct {
	Username string
	ClanID   string
}

type Session struct {
	ID        string
	User      User
	ExpiresAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

func (s *SessionStore) Create(user User) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateSessionID()
	session := &Session{
		ID:        id,
		User:      user,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	s.sessions[id] = session
	return session
}

func (s *SessionStore) Get(id string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil
	}
	if time.Now().After(session.ExpiresAt) {
		return nil
	}
	return session
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

const SessionCookieName = "tnrpt_session"

func ValidateCredentials(username, password string) (*User, bool) {
	username = strings.ToLower(strings.TrimSpace(username))
	password = strings.TrimSpace(password)

	if !strings.HasPrefix(username, "clan") {
		return nil, false
	}

	clanID := strings.TrimPrefix(username, "clan")
	if len(clanID) != 4 {
		return nil, false
	}

	expectedPassword := fmt.Sprintf("clan-%s", clanID)
	if password != expectedPassword {
		return nil, false
	}

	return &User{
		Username: username,
		ClanID:   clanID,
	}, true
}

func SetSessionCookie(w http.ResponseWriter, session *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func GetSessionFromRequest(r *http.Request, store *SessionStore) *Session {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}
	return store.Get(cookie.Value)
}
