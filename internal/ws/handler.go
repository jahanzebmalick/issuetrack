package ws

import (
	"issuetrack/internal/db"
	"issuetrack/internal/users"
	"log"
	"net/http"
	"strconv"
)

func ServeWS(w http.ResponseWriter, r *http.Request) {
	uid, ok := users.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "not logged in", http.StatusUnauthorized)
		return
	}
	pidStr := r.URL.Query().Get("project_id")
	projectID, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}
	var allowed bool
	err = db.Pool.QueryRow(r.Context(), `
	SELECT EXISTS (
	SELECT 1 FROM projects WHERE id = $1 AND owner_id = $2
	UNION 
	SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2)`,
		projectID, uid).Scan(&allowed)
	if err != nil {
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "no access to project", http.StatusForbidden)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		return
	}
	client := &Client{
		hub:       GlobalHub,
		conn:      conn,
		send:      make(chan []byte, 256),
		projectID: projectID,
		userID:    uid,
	}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
