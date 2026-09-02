package admin

import "context"

type contextKey int

const adminIDContextKey contextKey = 0

func withAdminID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, adminIDContextKey, id)
}

func adminIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(adminIDContextKey).(string)
	return id
}
