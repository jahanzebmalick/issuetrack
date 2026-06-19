package users

import (
	"context"
	"issuetrack/internal/db"
	"log"
	"net/http"
)

func setSession(ctx context.Context, sid string, userID int) {
	log.Printf("setSession called: sid=%s userID=%d", sid, userID)
	_, err := db.Pool.Exec(ctx, `
	INSERT INTO sessions (id, user_id) VALUES ($1, $2)`, sid, userID)
	if err != nil {
		log.Printf("setSession failed: %v", err)
	}

}
func deleteSession(ctx context.Context, sid string) {
	_, err := db.Pool.Exec(ctx, `
	DELETE FROM sessions WHERE id = $1`, sid)
	if err != nil {
		return
	}
}
func userIDFromCookie(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0, false
	}
	var userID int
	if err := db.Pool.QueryRow(r.Context(), `
	SELECT user_id FROM sessions WHERE id = $1`, cookie.Value).Scan(&userID); err != nil {
		return 0, false
	}
	return userID, true
}
