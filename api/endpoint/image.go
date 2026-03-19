package endpoint

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	tlpData "github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/utils"
)

var (
	imagePerPage = 24
)

func ImageCoordByTags(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tagIdStr := c.Param("tid")
		iPageStr := c.DefaultQuery(tlpData.ImagePageParam, "1")
		logg := logging.Enter(c, "page.public.tags", map[string]any{
			"tag_id":     tagIdStr,
			"image_page": iPageStr,
		})
		ret := data.APIResponse[[]data.ImageCoord]{}

		tagId, err := strconv.Atoi(tagIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid tag id"))
			ret.Error = utils.PtrString("invalid tag id")
			c.JSON(http.StatusBadRequest, ret)
			return
		}

		iPage, err := strconv.Atoi(iPageStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image_page"))
			ret.Error = utils.PtrString("invalid image page")
			c.JSON(http.StatusBadRequest, ret)
			return
		}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(c)

		images, err := dao.QueryImageCoordByTagACL(database, c, uint64(tagId), tpl.CreateDBOACL(acl))
		if err != nil {
			logging.ExitErr(logg, err)
			ret.Error = utils.PtrString("internal error")
			c.JSON(http.StatusInternalServerError, ret)
		}
		coords := make([]data.ImageCoord, 0, len(images))
		start := (iPage - 1) * imagePerPage
		end := start + imagePerPage
		generator := func(id uint64) string {
			return routes.CreateTagImagePath(uint64(tagId), id)
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

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", map[string]any{"coords": len(coords)})
	}
}
