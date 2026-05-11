package endpoint

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/logging"
	apiData "github.com/ignisVeneficus/lumenta/api/data"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"
)

func TagsQuery(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		view := ListView(c.DefaultQuery("view", "flat"))
		logg, ctx := logging.Enter(c.Request.Context(), "api/admin/tags", nil, map[string]any{
			"view": view,
		})
		ret := apiData.APIResponse[[]*apiData.Tag]{}

		if !view.IsValid() {
			logging.ExitErr(logg, fmt.Errorf("invalid view"))
			ret.HandleError("invalid view")
			c.AbortWithStatusJSON(http.StatusBadRequest, ret)
			return
		}

		database := db.GetDatabase()
		dboTags, err := dao.QueryTags(database, ctx)
		if err != nil {
			logging.ExitErr(logg, err)
			ret.HandleError("internal error")
			c.JSON(http.StatusInternalServerError, ret)
		}
		pointerTags := dbo.TagsToPointer(dboTags)
		tags := apiData.CreateTags(pointerTags)

		if view == ListViewTree {
			utils.SortByStringKey(tags, (*apiData.Tag).GetSorting)
			forest := data.BuildForest(tags)
			data.PopulatePath(forest)
			tags = forest.Roots
		} else {
			data.BuildFlatPath(tags)
		}

		ret.Data = tags
		ret.Status = apiData.StatuszOK

		c.IndentedJSON(http.StatusOK, ret)
		logging.Exit(logg, "ok", map[string]any{"tags": len(tags)})

	}
}
