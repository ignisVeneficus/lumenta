package endpoint

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/api"
	"github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/utils"
)

var (
	imagePerPage = 24
)

func ImageCoordByTags(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tagIdStr := c.Param("tid")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "1")
		logg, ctx := logging.Enter(c.Request.Context(), "api/public/images/coord/tags", tagIdStr, map[string]any{
			"tag_id":     tagIdStr,
			"image_page": iPageStr,
		})
		ret := data.APIResponse[[]data.ImageCoord]{}

		tagId, err := api.ParseTagID(tagIdStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid tag id"))
			ret.HandleError("invalid tag id")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		iPage, err := utils.ParseEmptyUint(iPageStr)
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
		start := int(iPage-1) * imagePerPage
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

func ImageCoordByAlbums(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		albumIdStr := c.Param("tid")
		iPageStr := c.DefaultQuery(routes.ImagePageParam, "1")
		logScope, ctx := logging.Enter(c.Request.Context(), "api/public/images/coord/album", albumIdStr, map[string]any{
			"album_id":   albumIdStr,
			"image_page": iPageStr,
		})
		ret := data.APIResponse[[]data.ImageCoord]{}

		albumID, err := api.ParseAlbumID(albumIdStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid album id"))
			ret.HandleError("invalid album id")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		iPage, err := utils.ParseEmptyUint(iPageStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid image_page"))
			ret.HandleError("invalid image page")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(ctx)

		images, err := dao.QueryImageCoordByAlbumDescendantIDsByACL(database, ctx, dbo.AlbumID(albumID), acl.ACLContext)
		if err != nil {
			logging.ExitErr(logScope, err)
			ret.HandleError("internal error")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
		}

		coords := make([]data.ImageCoord, 0, len(images))

		pagedImages, err := dao.QueryImageByAlbumACLPaged(database, ctx, dbo.AlbumID(albumID), acl.ACLContext, (iPage-1)*uint64(imagePerPage), uint64(imagePerPage))
		if err != nil {
			logging.ExitErr(logScope, err)
			ret.HandleError("internal error")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
			return
		}
		pagedImagesIds := make(map[dbo.ImageID]struct{})
		for _, img := range pagedImages {
			pagedImagesIds[*img.ID] = struct{}{}
		}

		generator := func(id uint64) string {
			return routes.CreateAlbumImagePath(routes.AlbumID(albumID), routes.ImageID(id))
		}
		for _, c := range images {
			if c.Latitude == nil || c.Longitude == nil {
				continue
			}
			coord := data.CreateImageCoord(c, generator)
			_, ok := pagedImagesIds[c.ID]
			if !ok {
				coord.Color = "muted"
			}
			coords = append(coords, coord)
		}
		ret.Data = coords
		ret.Status = data.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logScope, "ok", map[string]any{"coords": len(coords)})
	}
}
func ImageCoordByAlbumRoot(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		logScope, ctx := logging.Enter(c.Request.Context(), "api/public/images/coord/album/root", nil, nil)
		ret := data.APIResponse[[]data.ImageCoord]{}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(ctx)

		images, err := dao.QueryImageCoordByAlbumRootByACL(database, ctx, acl.ACLContext)
		if err != nil {
			logging.ExitErr(logScope, err)
			ret.HandleError("internal error")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
		}

		coords := make([]data.ImageCoord, 0, len(images))

		generator := func(id uint64) string {
			return routes.CreateImagePath(routes.ImageID(id))
		}
		for _, c := range images {
			if c.Latitude == nil || c.Longitude == nil {
				continue
			}
			coord := data.CreateImageCoord(c, generator)
			coord.Color = "muted"
			coords = append(coords, coord)
		}
		ret.Data = coords
		ret.Status = data.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logScope, "ok", map[string]any{"coords": len(coords)})
	}
}
