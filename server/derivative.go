package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config"
	derivativeConfig "github.com/ignisVeneficus/lumenta/config/derivative"
	"github.com/ignisVeneficus/lumenta/derivative"
	"github.com/ignisVeneficus/lumenta/logging"
)

func DerivativeHandler(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.MustGet(auth.AuthContextKey).(authData.ACLContext)
		logg := logging.Enter(c, "server.derivativeHandler", map[string]any{
			"id":   c.Param("id"),
			"type": c.Param("type"),
		})

		imgID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			logging.ExitErr(logg, err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid image id",
			})
			return
		}

		kind := c.Param("type")
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
		path, err := derivative.GetDerivativesPathWithACL(c, auth, imgID, *found, cfg.Filesystem)
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
			c.File(path)
			return
		}
		logging.ExitErr(logg, err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
}
