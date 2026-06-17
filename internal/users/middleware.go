package users

import (
	"context"
	"net/http"
)

type userIDkey struct{}

func WithUserID(ctx context.Context, uid int) context.Context {
	return context.WithValue(ctx, userIDkey{}, uid)
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	uid, ok := ctx.Value(userIDkey{}).(int)
	return uid, ok
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := userIDFromCookie(r)
		if !ok {
			http.Error(w, "not logged in", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), uid)))
	})
}
