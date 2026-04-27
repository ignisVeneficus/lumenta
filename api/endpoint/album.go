package endpoint

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/ignisVeneficus/lumenta/validate"
)

func AlbumQuery(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		view := ListView(c.DefaultQuery("view", "flat"))
		logg := logging.Enter(c, "api.admin.albums.get", map[string]any{
			"view": view,
		})
		ret := data.APIResponse[[]data.Album]{}

		if !view.IsValid() {
			logging.ExitErr(logg, fmt.Errorf("invalid view"))
			ret.HandleError("invalid `view`")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		database := db.GetDatabase()
		dboAlbums, err := dao.QueryAlbum(database, c)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("internal error")

			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
		}

		pointers := dbo.AlbumsToPointer(dboAlbums)
		fullName := utils.BuildPath(pointers)
		var dboAlbumTree []*dbo.Album
		if view == ListViewTree {
			dboAlbumTree = dbo.BuildAlbumsTree(pointers)
		} else {
			dboAlbumTree = dbo.BuildAlbumsFlat(pointers)
		}
		albums := data.CreateAlbums(dboAlbumTree, fullName)

		ret.Data = albums
		ret.Status = data.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", map[string]any{"albums": len(albums)})

	}
}

func mergeAlbumData(album dbo.Album, patch data.AlbumPatch) dbo.Album {
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
		var albumPatch data.AlbumPatch
		ret := data.APIStatusResponse{}
		albumIDStr := c.Param("id")
		c.BindJSON(&albumPatch)
		logg := logging.Enter(c, "page.admin.album.patch", map[string]any{
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
		album, err := dao.GetAlbumById(database, c, uint64(albumID))
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("album not found")
			c.AbortWithStatusJSON(http.StatusNotFound, ret)
			return
		}
		album = mergeAlbumData(album, albumPatch)

		dboGraph, err := dao.QueryAlbumGraph(database, c)
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
		err = dao.UpdateAlbum(database, c, album)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("cant save album")
			c.AbortWithStatusJSON(http.StatusInternalServerError, ret)
			return
		}
		ret.Status = data.StatuszOK
		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", nil)

	}
}
