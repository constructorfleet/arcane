package bootstrap

import (
	"bytes"
	"log/slog"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequestLogHandler(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
		t.Run("condenses slog-echo records at "+level.String(), func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(&requestLogHandler{
				handler: slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level}),
			})

			logger.LogAttrs(t.Context(), slog.LevelWarn, "Not Found",
				slog.Group("request",
					slog.Time("time", time.Now()), //nolint:sloglint // Mirrors the attribute shape slog-echo emits.
					slog.String("method", "GET"),
					slog.String("host", "localhost:3552"),
					slog.String("path", "/api/webhooks/trigger/fixture-path-token"),
					slog.String("query", "next="+url.QueryEscape("/api/webhooks/trigger/fixture-query-token")+"&page=2"),
					slog.Any("params", map[string]string{"id": "123"}),
					slog.String("route", ""),
					slog.String("referer", "https://example.com/api/webhooks/trigger/fixture-referer-token"),
					slog.Int("length", 0),
				),
				slog.Group("response",
					slog.Time("time", time.Now()), //nolint:sloglint // Mirrors the attribute shape slog-echo emits.
					slog.Duration("latency", 1500*time.Microsecond),
					slog.Int("status", 404),
					slog.Int("length", 24),
				),
				slog.Any("error", map[string]any{ //nolint:sloglint // Mirrors the attribute shape slog-echo emits.
					"code":     404,
					"message":  "Not Found",
					"internal": nil,
				}),
			)

			output := buf.String()
			assert.Contains(t, output, "method=GET")
			assert.Contains(t, output, "status=404")
			assert.Contains(t, output, "duration=")
			assert.Contains(t, output, `error="Not Found"`)
			assert.NotContains(t, output, "request.params")
			assert.Contains(t, output, "path=/api/webhooks/trigger/:token")
			assert.NotContains(t, output, "fixture-path-token")
			assert.NotContains(t, output, "fixture-query-token")
			assert.NotContains(t, output, "fixture-referer-token")
			if level == slog.LevelDebug {
				assert.Contains(t, output, `query="next=/api/webhooks/trigger/:token&page=2"`)
				assert.Contains(t, output, "referer=https://example.com/api/webhooks/trigger/:token")
			} else {
				assert.NotContains(t, output, "query=")
				assert.NotContains(t, output, "referer=")
			}
			assert.NotContains(t, output, "map[code:")
		})
	}

	t.Run("passes plain records through unchanged", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(&requestLogHandler{
			handler: slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}),
		})

		logger.InfoContext(t.Context(), "plain record", "component", "scheduler")

		assert.Contains(t, buf.String(), "msg=\"plain record\" component=scheduler")
	})
}
