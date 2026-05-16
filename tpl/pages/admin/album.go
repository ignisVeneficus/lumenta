package admin

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/definitions"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/ruleengine"
	"github.com/ignisVeneficus/lumenta/server/routes"
	"github.com/ignisVeneficus/lumenta/tpl"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	adminData "github.com/ignisVeneficus/lumenta/tpl/data/admin"
	"github.com/ignisVeneficus/lumenta/tpl/pages"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/ignisVeneficus/lumenta/validate"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func createAlbumBreadcrumbs(lastItem string, loc string, i18n *i18n.Service) tplData.Breadcrumbs {
	return tplData.Breadcrumbs{
		tpl.GetAdminMain(loc, i18n),
		tplData.Breadcrumb{
			Label: i18n.T(loc, "nav.page.admin.albums.short", nil),
			Link:  template.URL(routes.CreateAdminAlbumsPath()),
			Type:  "page",
			Title: i18n.T(loc, "nav.page.admin.albums.label", nil),
		},
		tplData.Breadcrumb{
			Label: lastItem,
			Type:  "Edit",
		},
	}
}

func newAlbumPageContext(album dbo.Album) adminData.AlbumContext {
	state := adminData.StateNew
	if album.ID != nil {
		state = adminData.StateEdit
	}
	return adminData.AlbumContext{
		AlbumForm: adminData.AlbumForm{
			DBOID:       album.ID,
			ID:          tpl.AlbumIDToString(album.ID),
			Name:        album.Name,
			Description: utils.FromStringPtr(album.Description),
			ParentID:    tpl.AlbumIDToString(album.ParentID),
			RuleJSON:    string(template.JS(string(album.RuleJSON))),
			ACLLevel:    strconv.FormatUint(uint64(album.ACLLevel), 10),
			ACLUserID:   strconv.FormatUint(uint64(album.ACLUserID), 10),
			Rank:        tpl.UintToString(&album.Rank),
		},
		CoverImage: (*routes.ImageID)(album.CoverImageID),
		State:      state,
	}
}

func toDBAlbum(album dbo.Album, form adminData.AlbumForm) (dbo.Album, validate.ValidationErrors) {
	// only conversions errors
	validateErrors := make(validate.ValidationErrors)

	album.Name = form.Name
	parent, err := utils.StrToPtrUint64(form.ParentID)
	if err != nil {
		validateErrors.AddError(definitions.AlbumFieldParentID, "Not an integer number")
	} else {
		album.ParentID = (*dbo.AlbumID)(parent)
	}

	if form.Description == "" {
		album.Description = nil
	} else {
		album.Description = &form.Description
	}
	album.RuleJSON = json.RawMessage(form.RuleJSON)
	aclLevel, err := utils.StrToUint64(form.ACLLevel)
	if err != nil {
		validateErrors.AddError(definitions.AlbumFieldACLLevel, "Not an integer number")
	} else {
		album.ACLLevel = dbo.DBACLLevel(aclLevel)
	}

	var userId uint64 = 0
	if album.ACLLevel == dbo.DBACLLevelUser {
		userId, err = utils.StrToUint64(form.ACLUserID)
		if err == nil {
			validateErrors.AddError(definitions.AlbumFieldACLUserID, "Not an integer number")
		}
	}
	album.ACLUserID = dbo.UserID(userId)

	return album, validateErrors
}

func handleAlbumForm(c *gin.Context, album dbo.Album, save func(dbo.Album) (dbo.AlbumID, error), graph []*dbo.AlbumGraph) (*adminData.AlbumContext, error) {
	form := newAlbumPageContext(album)

	if c.Request.Method == http.MethodPost {

		c.ShouldBind(&form)
		// TODO: clean (sanity) response
		log.Logger.Debug().Object("form", logging.WithLevel(zerolog.DebugLevel, &form)).Msg("readed")
		form.State = "validate"

		newAlbum, errors := toDBAlbum(album, form.AlbumForm)
		if errors.HasErrors() {
			form.Errors = errors
		} else {
			form.Errors = validate.ValidateAlbum(newAlbum, graph)
			if errors.HasErrors() {
				form.Errors = errors
			}
		}
		if !form.Errors.HasErrors() {

			log.Logger.Debug().Object("album", logging.WithLevel(zerolog.TraceLevel, &newAlbum)).Msg("converted")
			//log.Logger.Debug().Str("rule", string(newAlbum.RuleJSON)).Msg("converted")

			id, err := save(newAlbum)
			if err != nil {
				return &form, err
			}
			form.DBOID = &id
			form.State = "saved"
			return &form, nil
		}
	}

	return &form, nil
}

func NewAlbumPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/album/new", nil, nil)
		database := db.GetDatabase()
		RuleJson, _ := json.Marshal(ruleengine.CreateEmptyRuleGroup())
		album := dbo.Album{
			ACLLevel: dbo.DBACLLevelPublic,
			Rank:     0,
			RuleJSON: RuleJson,
		}
		graph, err := dao.QueryAlbumGraph(database, ctx)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		albumCtx, err := handleAlbumForm(
			c,
			album,
			func(a dbo.Album) (dbo.AlbumID, error) {
				return dao.CreateAlbum(database, ctx, &a)
			},
			dbo.AlbumGraphToPointer(graph),
		)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if albumCtx.State == adminData.StateSaved {
			logging.Exit(logScope, "redirect", nil)
			url := routes.BuildAdminAlbumPath(routes.AlbumID(*albumCtx.DBOID))
			url.WithParam(routes.QueryFlash, string(adminData.FlashCreated))

			c.Redirect(http.StatusSeeOther, url.String())
			return
		}
		albumPageCtx := adminData.AlbumPageContext{}
		pageCtx := albumPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "album", tplData.SurfaceAdmin)

		albumPageCtx.Album = *albumCtx
		albumPageCtx.Breadcrumbs = createAlbumBreadcrumbs("New Album", loc, i18n)
		if err := r.RenderPage(c.Writer, "admin/album", albumPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)

	}
}

func EditAlbumPage(r *tpl.TemplateResolver, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		i18n := i18n.Get()
		loc := tpl.L(c)
		albumIDStr := c.Param("id")
		flash := c.Query(routes.QueryFlash)
		logScope, ctx := logging.Enter(c.Request.Context(), "server/page/admin/album/edit", albumIDStr, map[string]any{
			"album": albumIDStr,
			"flash": flash,
		})
		albumID, err := tpl.ParseAlbumID(albumIDStr)
		if err != nil {
			logging.ExitErr(logScope, fmt.Errorf("invalid album Id"))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid album Id"})
			return
		}

		database := db.GetDatabase()
		album, err := dao.GetAlbumByID(database, c, dbo.AlbumID(albumID))
		if err != nil {
			logging.ExitErr(logScope, err)
			pages.Soft404(r, cfg, c, tplData.SurfacePublic, "tag", routes.CreateAdminAlbumsPath(), uint64(albumID))
			return
		}
		graph, err := dao.QueryAlbumGraph(database, ctx)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		name := album.Name
		albumCtx, err := handleAlbumForm(
			c,
			album,
			func(a dbo.Album) (dbo.AlbumID, error) {
				return dbo.AlbumID(albumID), dao.UpdateAlbum(database, ctx, a)
			},
			dbo.AlbumGraphToPointer(graph),
		)
		if err != nil {
			logging.ExitErr(logScope, err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		if albumCtx.State == adminData.StateSaved {
			logging.Exit(logScope, "redirect", nil)
			url := routes.BuildAdminAlbumPath(routes.AlbumID(*albumCtx.DBOID))
			url.WithParam(routes.QueryFlash, string(adminData.FlashSaved))

			c.Redirect(http.StatusSeeOther, url.String())
			return
		}
		albumPageCtx := adminData.AlbumPageContext{}
		pageCtx := albumPageCtx.GetPage()
		tpl.CreatePageContext(pageCtx, cfg, c, "album", tplData.SurfaceAdmin)
		albumCtx.Flash = adminData.Flash(flash)
		albumPageCtx.Album = *albumCtx
		albumPageCtx.Breadcrumbs = createAlbumBreadcrumbs("Edit: "+name, loc, i18n)

		if err := r.RenderPage(c.Writer, "admin/album", albumPageCtx, loc, i18n); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			logging.ExitErr(logScope, err)
			return
		}
		logging.Exit(logScope, "ok", nil)
	}
}
