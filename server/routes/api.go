package routes

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/tpl/data"
)

const (
	ApiPrefix  = "/api"
	apiTagPath = "/tags/%d"
)

func GetApiTagPath() string {
	return getPath(apiTagPath, ":tid")
}
func CreateApiTagPath(id uint64) string {
	return fmt.Sprintf(ApiPrefix+apiTagPath, id)
}
func BuildApiTagPath(tagId uint64) *data.URLBuilder {
	return data.NewURL(CreateApiTagPath(tagId))
}
