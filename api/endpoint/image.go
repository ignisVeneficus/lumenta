package endpoint

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/server/routes"
)

var (
	imagePerPage = 24
)

func ImageCoordByTags(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tagIdStr := c.Param("tid")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "1")
		logg, ctx := logging.Enter(c.Request.Context(), "api/public/images/tags", tagIdStr, map[string]any{
			"tag_id":     tagIdStr,
			"image_page": iPageStr,
		})
		ret := data.APIResponse[[]data.ImageCoord]{}

		tagId, err := strconv.Atoi(tagIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid tag id"))
			ret.HandleError("invalid tag id")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		iPage, err := strconv.Atoi(iPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image_page"))
			ret.HandleError("invalid image page")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(ctx)

		images, err := dao.QueryImageCoordByTagByACL(database, ctx, dbo.TagID(tagId), acl.ACLContext)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("internal error")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
		}
		coords := make([]data.ImageCoord, 0, len(images))
		start := (iPage - 1) * imagePerPage
		end := start + imagePerPage
		uintTagID := uint64(tagId)
		generator := func(id uint64) string {
			return routes.CreateTagImagePath(routes.TagID(uintTagID), routes.ImageID(id))
		}
		for i, c := range images {
			if c.Latitude == nil || c.Longitude == nil {
				continue
			}
			coord := data.CreateImageCoord(c, generator)
			if i < start || i > end {
				coord.Color = "muted"
			}
			coords = append(coords, coord)
		}
		ret.Data = coords
		ret.Status = data.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", map[string]any{"coords": len(coords)})
	}
}
