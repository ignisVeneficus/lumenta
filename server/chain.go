package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
