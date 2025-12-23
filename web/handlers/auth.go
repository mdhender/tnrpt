// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package handlers

import (
	"context"
	"net/http"

	"github.com/mdhender/tnrpt"
	"github.com/mdhender/tnrpt/web/auth"
	"github.com/mdhender/tnrpt/web/templates"
)

// withUsername adds the username to the request context.
// Uses string key "username" to match template ctx.Value("username") checks.
func withUsername(r *http.Request, username string) *http.Request {
	ctx := context.WithValue(r.Context(), "username", username)
	return r.WithContext(ctx)
}

func (h *Handlers) LoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if session := auth.GetSessionFromRequest(r, h.sessions); session != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	data := templates.LayoutData{Version: tnrpt.Version().String()}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.LoginPage("", data).Render(r.Context(), w); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := templates.LayoutData{Version: tnrpt.Version().String()}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Invalid form submission", data).Render(r.Context(), w)
		return
	}

	handle := r.FormValue("username")
	password := r.FormValue("password")
	gameID := r.FormValue("game")
	if gameID == "" {
		gameID = "0301" // default game
	}

	user, err := h.store.ValidateCredentials(r.Context(), handle, password, gameID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Authentication error", data).Render(r.Context(), w)
		return
	}
	if user == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		templates.LoginPage("Invalid username or password", data).Render(r.Context(), w)
		return
	}

	session := h.sessions.Create(*user)
	auth.SetSessionCookie(w, session)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil {
		h.sessions.Delete(cookie.Value)
	}
	auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handlers) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := auth.GetSessionFromRequest(r, h.sessions)
		if session == nil {
			if h.autoAuthUser != nil {
				session = h.sessions.Create(*h.autoAuthUser)
				auth.SetSessionCookie(w, session)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
		}
		r = withUsername(r, session.User.Handle)
		next(w, r)
	}
}
