package controller

import (
	"context"
	"net"
	"time"
)

type requestConnectionContextKey struct{}

// WithRequestConnection stores the accepted socket in the request context so
// the few intentionally long streaming routes can override the panel's strict
// default HTTP deadlines without weakening ordinary requests.
func WithRequestConnection(ctx context.Context, connection net.Conn) context.Context {
	return context.WithValue(ctx, requestConnectionContextKey{}, connection)
}

func extendRequestConnectionDeadline(ctx context.Context, duration time.Duration) bool {
	connection, ok := ctx.Value(requestConnectionContextKey{}).(net.Conn)
	if !ok || connection == nil || duration <= 0 {
		return false
	}
	deadline := time.Now().Add(duration)
	return connection.SetReadDeadline(deadline) == nil && connection.SetWriteDeadline(deadline) == nil
}
