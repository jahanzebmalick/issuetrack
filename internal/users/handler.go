package users

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"issuetrack/internal/db"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type meResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

func newSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
func SignupHandler(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "usernama and password required", http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "hash failed", http.StatusInternalServerError)
		return
	}
	_, err = db.Pool.Exec(r.Context(),
		`INSERT INTO users (username, password_hash) VALUES ($1, $2)`, req.Username, string(hash))

	if err != nil {
		http.Error(w, "username already taken", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}
	var userID int
	var storedhash string
	err := db.Pool.QueryRow(r.Context(), `
	SELECT id, password_hash FROM users WHERE username = $1`,
		req.Username).Scan(&userID, &storedhash)
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedhash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	sid := newSessionID()
	setSession(sid, userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	})
}
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		deleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	})
}
func MeHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := UserIDFromContext(r.Context())
	var u struct {
		ID        int     `json:"id"`
		Username  string  `json:"username"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := db.Pool.QueryRow(r.Context(), `
		SELECT id , username, avatar_url FROM users WHERE id = $1`, userID,
	).Scan(&u.ID, &u.Username, &u.AvatarURL); err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&u)
}
