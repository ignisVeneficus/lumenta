package endpoint

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/utils"
)

func TagsQuery(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		view := ListView(c.DefaultQuery("view", "flat"))
		logg := logging.Enter(c, "api.admin.tags", map[string]any{
			"view": view,
		})
		ret := data.APIResponse[[]data.Tag]{}

		if !view.IsValid() {
			logging.ExitErr(logg, fmt.Errorf("invalid view"))
			ret.HandleError("invalid view")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		database := db.GetDatabase()
		dboTags, err := dao.QueryTags(database, c)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("internal error")
			c.JSON(http.StatusInternalServerError, ret)
		}
		pointerTags := dbo.TagsToPointer(dboTags)
		fullName := utils.BuildPath(pointerTags)
		var dboTagTree dbo.TagsTree
		if view == ListViewTree {
			dboTagTree = dbo.BuildTagsTree(dbo.TagsToPointer(dboTags))
		} else {
			dboTagTree = dbo.BuildTagsFlat(dbo.TagsToPointer(dboTags))
		}
		tags := data.CreateTags(dboTagTree, fullName)

		ret.Data = tags
		ret.Status = data.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", map[string]any{"tags": len(tags)})

	}
}
