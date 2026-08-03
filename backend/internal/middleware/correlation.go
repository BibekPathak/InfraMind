package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey string

const CorrelationIDKey ctxKey = "correlation_id"

// CorrelationID ensures every request carries a correlation_id, either
// forwarded from an inbound header (for trace propagation) or generated.
func CorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			id = uuid.NewString()
		}

		w.Header().Set("X-Correlation-ID", id)
		ctx := context.WithValue(r.Context(), CorrelationIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CorrelationIDFrom returns the correlation id from context, or empty.
func CorrelationIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(CorrelationIDKey).(string)
	return id
}
