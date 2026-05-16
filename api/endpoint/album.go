package endpoint

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	apiData "github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/ignisVeneficus/lumenta/validate"
)

func AlbumQuery(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		view := ListView(c.DefaultQuery("view", "flat"))
		logg, ctx := logging.Enter(c.Request.Context(), "api/admin/albums/get", nil, map[string]any{
			"view": view,
		})
		ret := apiData.APIResponse[[]*apiData.Album]{}

		if !view.IsValid() {
			logging.ExitErr(logg, fmt.Errorf("invalid view"))
			ret.HandleError("invalid `view`")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		database := db.GetDatabase()
		dboAlbums, err := dao.QueryAlbum(database, ctx)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("internal error")

			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
		}

		pointers := dbo.AlbumsToPointer(dboAlbums)

		albums := apiData.CreateAlbums(pointers)

		if view == ListViewTree {
			utils.SortByStringKey(albums, (*apiData.Album).GetSorting)
			forest := data.BuildForest(albums)
			data.PopulatePath(forest)
			albums = forest.Roots
		} else {
			data.BuildFlatPath(albums)
		}

		ret.Data = albums
		ret.Status = apiData.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", map[string]any{"albums": len(albums)})

	}
}

func mergeAlbumData(album dbo.Album, patch apiData.AlbumPatch) dbo.Album {
	patch.Name.Apply(&album.Name)
	patch.Description.ApplyPtr(&album.Description)
	patch.ParentID.ApplyPtr(&album.ParentID)
	patch.Rank.Apply(&album.Rank)
	patch.CoverImageID.ApplyPtr(&album.CoverImageID)
	patch.ACLLevel.Apply(&album.ACLLevel)
	patch.ACLUserID.Apply(&album.ACLUserID)
	return album
}

func AlbumPatch(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var albumPatch apiData.AlbumPatch
		ret := apiData.APIStatusResponse{}
		albumIDStr := c.Param("id")
		c.BindJSON(&albumPatch)
		logg, ctx := logging.Enter(c.Request.Context(), "page/admin/album/patch", albumIDStr, map[string]any{
			"ID":    albumIDStr,
			"album": albumPatch,
		})
		albumID, err := strconv.Atoi(albumIDStr)
		if err != nil {
			logging.ExitErr(logg, fmt.Errorf("invalid album Id"))
			ret.HandleError("invalid album Id")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}
		database := db.GetDatabase()
		album, err := dao.GetAlbumByID(database, ctx, dbo.AlbumID(albumID))
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("album not found")
			c.AbortWithStatusJSON(http.StatusNotFound, ret)
			return
		}
		album = mergeAlbumData(album, albumPatch)

		dboGraph, err := dao.QueryAlbumGraph(database, ctx)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("cant query albums")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
			return
		}

		graph := dbo.AlbumGraphToPointer(dboGraph)
		errors := validate.ValidateAlbum(album, graph)

		if errors.HasErrors() {
			logging.ExitErrParams(logg, fmt.Errorf("validation error"), map[string]any{
				"errors": errors,
			})
			ret.HandleValidateErrors(errors)
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}
		err = dao.UpdateAlbum(database, ctx, album)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("cant save album")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
			return
		}
		ret.Status = apiData.StatuszOK
		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", nil)

	}
}
