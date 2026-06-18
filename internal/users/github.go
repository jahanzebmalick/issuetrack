package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"issuetrack/internal/db"
)

func GitHubLogin(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if clientID == "" {
		http.Error(w, "github oauth not configured", http.StatusServiceUnavailable)
		return
	}

	state := uuid.New().String()
	http.SetCookie(w, &http.Cookie{
		Name:     "gh_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		MaxAge:   600,
	})

	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&scope=read:user,repo&state=%s",
		clientID, state,
	)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func GitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	stateCookie, err := r.Cookie("gh_state")
	if err != nil || stateCookie.Value != state {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)

	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(r.Context(), "POST",
		"https://github.com/login/oauth/access_token",
		strings.NewReader(form.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var tokenData struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenData)
	if tokenData.AccessToken == "" {
		http.Error(w, "no access token", http.StatusInternalServerError)
		return
	}

	userReq, _ := http.NewRequestWithContext(r.Context(), "GET",
		"https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	userReq.Header.Set("Accept", "application/vnd.github+json")

	userResp, err := client.Do(userReq)
	if err != nil {
		http.Error(w, "user fetch failed", http.StatusInternalServerError)
		return
	}
	defer userResp.Body.Close()

	var ghUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		Name      string `json:"name"`
	}
	json.NewDecoder(userResp.Body).Decode(&ghUser)
	if ghUser.ID == 0 {
		http.Error(w, "github user has no id", http.StatusInternalServerError)
		return
	}

	var userID int
	err = db.Pool.QueryRow(r.Context(),
		`SELECT id FROM users WHERE github_id = $1`,
		ghUser.ID,
	).Scan(&userID)

	if err == pgx.ErrNoRows {
		candidate := ghUser.Login
		for i := 2; ; i++ {
			var exists int
			db.Pool.QueryRow(r.Context(),
				`SELECT COUNT(*) FROM users WHERE username = $1`, candidate).Scan(&exists)
			if exists == 0 {
				break
			}
			candidate = fmt.Sprintf("%s-gh%d", ghUser.Login, i)
		}

		err = db.Pool.QueryRow(r.Context(),
			`INSERT INTO users (username, github_id, github_username, github_access_token, avatar_url)
				 VALUES ($1, $2, $3, $4, $5)
				 RETURNING id`,
			candidate, ghUser.ID, ghUser.Login, tokenData.AccessToken, ghUser.AvatarURL,
		).Scan(&userID)

	} else if err == nil {
		_, err = db.Pool.Exec(r.Context(),
			`UPDATE users SET github_access_token=$1, avatar_url=$2 WHERE id=$3`,
			tokenData.AccessToken, ghUser.AvatarURL, userID,
		)
	}
	if err != nil {
		http.Error(w, "user save failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sid := uuid.New().String()
	setSession(sid, userID)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
	})

	frontend := os.Getenv("FRONTEND_URL")
	if frontend == "" {
		frontend = "https://issue-track-frontend.vercel.app"
	}
	http.Redirect(w, r, frontend, http.StatusFound)
}
