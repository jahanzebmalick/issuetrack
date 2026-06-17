package main

import (
	"context"
	"issuetrack/internal/activities"
	"issuetrack/internal/attachments"
	"issuetrack/internal/comments"
	"issuetrack/internal/db"
	"issuetrack/internal/issues"
	"issuetrack/internal/members"
	"issuetrack/internal/projects"
	"issuetrack/internal/tags"
	"issuetrack/internal/users"
	"issuetrack/internal/ws"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
	ctx := context.Background()
	_ = os.MkdirAll("storage", 0755)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://issuetrack:dev_password@localhost:5434/issuetrack"
	}
	if err := db.Init(ctx, dsn); err != nil {
		log.Fatal("db init;", err)
	}
	if err := db.RunMigrations(ctx, "migrations"); err != nil {
		log.Fatal("migrations:", err)
	}
	projStore := projects.NewStore(db.Pool)
	projHandlers := projects.NewHandlers(projStore)

	issueStore := issues.NewStore(db.Pool)
	issueHandlers := issues.NewHandlers(issueStore)

	commentStore := comments.NewStore(db.Pool)
	commentHandlers := comments.NewHandlers(commentStore)

	tagsStore := tags.NewStore(db.Pool)
	tagHandlers := tags.NewHandlers(tagsStore)

	membersStore := members.NewStore(db.Pool)
	memberHandler := members.NewHandlers(membersStore)

	ws.GlobalHub = ws.NewHub()
	go ws.GlobalHub.Run()

	attStore := attachments.NewStore(db.Pool)
	attHandlers := attachments.NewHandlers(attStore)

	activities.GlobalStore = activities.NewStore(db.Pool)
	actHandlers := activities.NewHandlers(activities.GlobalStore)

	r := chi.NewRouter()
	r.Route("/api", func(r chi.Router) {
		r.Post("/signup", users.SignupHandler)
		r.Post("/login", users.LoginHandler)
		r.Post("/logout", users.LogoutHandler)

		r.Group(func(r chi.Router) {
			r.Use(users.RequireAuth)
			r.Post("/projects", projHandlers.Create)
			r.Get("/projects", projHandlers.ListMine)
			r.Get("/projects/{id}", projHandlers.Get)
			r.Patch("/projects/{id}", projHandlers.Update)
			r.Delete("/projects/{id}", projHandlers.Delete)

			r.Post("/projects/{projectId}/issues", issueHandlers.Create)
			r.Get("/projects/{projectId}/issues", issueHandlers.ListByProject)
			r.Get("/issues/{id}", issueHandlers.Get)
			r.Patch("/issues/{id}", issueHandlers.Update)
			r.Delete("/issues/{id}", issueHandlers.Delete)

			r.Post("/issues/{issueId}/comments", commentHandlers.Create)
			r.Get("/issues/{issueId}/comments", commentHandlers.ListByIssue)
			r.Delete("/comments/{id}", commentHandlers.Delete)

			r.Post("/projects/{projectId}/tags", tagHandlers.Create)
			r.Get("/projects/{projectId}/tags", tagHandlers.ListByProject)
			r.Delete("/tags/{id}", tagHandlers.Delete)

			r.Post("/issues/{issueId}/tags", tagHandlers.Attach)
			r.Delete("/issues/{issueId}/tags/{tagId}", tagHandlers.Detach)
			r.Get("/issues/{issueId}/tags", tagHandlers.ListByIssue)

			r.Post("/projects/{projectId}/members", memberHandler.Invite)
			r.Get("/projects/{projectId}/members", memberHandler.ListByProject)
			r.Patch("/projects/{projectId}/members/{userId}", memberHandler.UpdateRole)
			r.Delete("/projects/{projectId}/members/{userId}", memberHandler.Remove)

			r.Get("/ws", ws.ServeWS)

			r.Post("/issues/{issueId}/attachments", attHandlers.Upload)
			r.Get("/issues/{issueId}/attachments", attHandlers.ListByIssue)
			r.Get("/attachments/{id}", attHandlers.Download)
			r.Delete("/attachments/{id}", attHandlers.Delete)

			r.Get("/projects/{projectId}/activity", actHandlers.ListByProject)
		})
	})
	log.Println("server starting on: 8080")
	http.ListenAndServe(":8080", r)

}
