package endpoint

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/api"
	apiData "github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/ignisVeneficus/lumenta/validate"
	"github.com/rs/zerolog/log"
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
		ret := apiData.APIResponse[[]apiData.ImageCoord]{}

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
		coords := make([]apiData.ImageCoord, 0, len(images))
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
			coord := apiData.CreateImageCoord(c, generator)
			if i < start || i > end {
				coord.Color = "muted"
			}
			coords = append(coords, coord)
		}
		ret.Data = coords
		ret.Status = apiData.StatuszOK

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
		ret := apiData.APIResponse[[]apiData.ImageCoord]{}

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

		coords := make([]apiData.ImageCoord, 0, len(images))

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
			coord := apiData.CreateImageCoord(c, generator)
			_, ok := pagedImagesIds[c.ID]
			if !ok {
				coord.Color = "muted"
			}
			coords = append(coords, coord)
		}
		ret.Data = coords
		ret.Status = apiData.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logScope, "ok", map[string]any{"coords": len(coords)})
	}
}
func ImageCoordByAlbumRoot(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		logScope, ctx := logging.Enter(c.Request.Context(), "api/public/images/coord/album/root", nil, nil)
		ret := apiData.APIResponse[[]apiData.ImageCoord]{}

		database := db.GetDatabase()
		acl := auth.GetAuthContex(ctx)

		images, err := dao.QueryImageCoordByAlbumRootByACL(database, ctx, acl.ACLContext)
		if err != nil {
			logging.ExitErr(logScope, err)
			ret.HandleError("internal error")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
		}

		coords := make([]apiData.ImageCoord, 0, len(images))

		generator := func(id uint64) string {
			return routes.CreateImagePath(routes.ImageID(id))
		}
		for _, c := range images {
			if c.Latitude == nil || c.Longitude == nil {
				continue
			}
			coord := apiData.CreateImageCoord(c, generator)
			coord.Color = "muted"
			coords = append(coords, coord)
		}
		ret.Data = coords
		ret.Status = apiData.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logScope, "ok", map[string]any{"coords": len(coords)})
	}
}

func ImagePatch(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var imagePatch apiData.ImagePatch
		ret := apiData.APIStatusResponse{}
		imageIDStr := c.Param("id")
		c.BindJSON(&imagePatch)
		logg, ctx := logging.Enter(c.Request.Context(), "page/admin/image/patch", imageIDStr, map[string]any{
			"ID":    imageIDStr,
			"image": imagePatch,
		})
		imageID, err := strconv.Atoi(imageIDStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid image Id"))
			ret.HandleError("invalid image Id")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}
		database := db.GetDatabase()
		image, err := dao.GetImageByID(database, ctx, dbo.ImageID(imageID))
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("image not found")
			c.AbortWithStatusJSON(http.StatusNotFound, ret)
			return
		}
		image = mergeImageData(image, imagePatch)
		if imagePatch.FocusMode.Set {
			image.FocusSource = dbo.ValueSourceUser
		}
		if imagePatch.ACLLevel.Set || imagePatch.ACLUserID.Set {
			image.ACLSource = dbo.ValueSourceUser
		}
		errors := validate.ValidateImage(image)

		if errors.HasErrors() {
			logging.ExitErrParams(logg, fmt.Errorf("validation error"), map[string]any{
				"errors": errors,
			})
			ret.HandleValidateErrors(errors)
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}
		err = dao.PatchImage(database, ctx, image)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("cant save image")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
			return
		}
		ret.Status = apiData.StatuszOK
		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", nil)

	}
}
func mergeImageData(image dbo.Image, patch apiData.ImagePatch) dbo.Image {
	patch.ACLLevel.Apply(&image.ACLLevel)
	log.Logger.Warn().Any("ACL", image.ACLLevel).Any("patch", patch.ACLLevel).Msg("acl")
	patch.ACLUserID.Apply(&image.ACLUserID)
	patch.FocusMode.Apply(&image.FocusMode)
	patch.FocusX.Apply(image.FocusX)
	patch.FocusY.Apply(image.FocusY)
	return image
}
