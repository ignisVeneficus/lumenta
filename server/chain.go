package server

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const RequestIDKey = "request_id"
const RequestIDHeader = "X-Request-Id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(RequestIDHeader)

		if reqID == "" {
			reqID = uuid.NewString()
		}

		// contextbe
		c.Set(RequestIDKey, reqID)
		//logging
		c.Set(logging.TraceIDKey, reqID)

		// response headerbe
		c.Writer.Header().Set(RequestIDHeader, reqID)

		c.Next()
	}
}

func getLogEvent(status int, error string) *zerolog.Event {
	switch {
	case status >= 400 && status < 500:
		return log.Logger.Warn()
	case status >= 500:
		return log.Logger.Error().Str("error", error)
	default:
		return log.Logger.Info()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		reqID, _ := c.Get(RequestIDKey)

		logEvent := getLogEvent(c.Writer.Status(), c.Errors.String())
		logEvent.Str(logging.FieldFunc, "server.request").
			Str(logging.FieldEvent, "msg.response").Int(logging.FieldResult, c.Writer.Status())
		d := zerolog.Dict()
		d.Any("request_id", reqID).
			Str("path", c.Request.URL.Path).
			Str("method", c.Request.Method).
			Dur("latency", latency)
		logEvent.Dict(logging.FieldParams, d)
		logEvent.Msg("")
	}
}

func BrowserCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		// API alapértelmezés: user-specifikus
		c.Header("Cache-Control", "private")
		c.Header("Vary", "Authorization")
		c.Next()
	}
}

func SecureHeaders(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {

		h := c.Writer.Header()
		// clickjacking
		h.Set("X-Frame-Options", "DENY")
		// mime sniffing
		h.Set("X-Content-Type-Options", "nosniff")
		// XSS filter (legacy)
		h.Set("X-XSS-Protection", "0")
		// referrer policy
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// permissions policy
		h.Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=()")
		if cfg.Env == config.EnvProduction {
			// HSTS
			h.Set("Strict-Transport-Security",
				"max-age=63072000; includeSubDomains; preload")
		}
		// CSP (nagyon fontos)
		h.Set("Content-Security-Policy", buildCSP(cfg))
		c.Next()
	}
}
func buildCSP(cfg config.Config) string {
	var directives []string

	directives = append(directives, "default-src 'self'")
	directives = append(directives, "base-uri 'self'")
	directives = append(directives, "object-src 'none'")
	directives = append(directives, "frame-ancestors 'none'")

	directives = append(directives, "img-src 'self' data: blob:")

	/* TODO: config
	scriptSrc := []string{"'self'"}
	scriptSrc = append(scriptSrc, cfg.ScriptSrc...)
	directives = append(directives, "script-src "+strings.Join(scriptSrc, " "))
	*/

	/* TODO: config
	styleSrc := []string{"'self'"}
	styleSrc = append(styleSrc, cfg.StyleSrc...)
	directives = append(directives, "style-src "+strings.Join(styleSrc, " "))
	*/
	/* TODO: config
	fontSrc := []string{"'self'"}
	fontSrc = append(fontSrc, cfg.FontSrc...)
	directives = append(directives, "font-src "+strings.Join(fontSrc, " "))
	*/
	/* TODO: config analytics / API / endpoints
	connectSrc := []string{"'self'"}
	connectSrc = append(connectSrc, cfg.ConnectSrc...)
	directives = append(directives, "connect-src "+strings.Join(connectSrc, " "))
	*/
	return strings.Join(directives, "; ")
}
