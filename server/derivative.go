package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config"
	derivativeConfig "github.com/ignisVeneficus/lumenta/config/derivative"
	"github.com/ignisVeneficus/lumenta/derivative"
)

func DerivativeHandler(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.MustGet(auth.AuthContextKey).(authData.ACLContext)
		imageIdStr := c.Param("id")
		kind := c.Param("type")

		logg, ctx := logging.Enter(c.Request.Context(), "server/derivativeHandler", imageIdStr, map[string]any{
			"id":   imageIdStr,
			"type": kind,
		})

		imgID, err := strconv.ParseUint(imageIdStr, 10, 64)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid image id",
			})
			return
		}

		derivativesCfx := cfg.Derivatives
		var found *derivativeConfig.DerivativeConfig
		for _, d := range derivativesCfx {
			if d.Postfix == kind {
				found = &d
				break
			}
		}
		if found == nil {
			logging.ExitErr(logg, fmt.Errorf("invalid type"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid type",
			})
			return
		}
		path, err := derivative.GetDerivativesPathWithACL(ctx, auth, imgID, *found, cfg.Filesystem)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		_, err = os.Stat(path)
		if os.IsNotExist(err) {
			logging.Exit(logg, "generating", nil)
			c.Header("Retry-After", "1")
			c.AbortWithStatus(http.StatusAccepted)
			return
		}
		if err == nil {
			logging.Exit(logg, "ok", nil)
			c.Header("Content-Type", "image/jpeg")
			c.Header("Cache-Control", "public, max-age=31536000")
			c.File(path)
			return
		}
		logging.ExitErr(logg, err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}
func DefaultHTMLMime() gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Next()

		h := c.Writer.Header()

		if h.Get("Content-Type") == "" {
			h.Set("Content-Type", "text/html; charset=utf-8")
		}
	}
}
