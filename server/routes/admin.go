package routes

import (
	"fmt"

	"github.com/ignisVeneficus/lumenta/tpl/data"
)

const (
	adminPrefix  = "/admin"
	adminRoot    = "/"
	adminFsPath  = "/fs/%s"
	adminImgPath = "/img/%d"
)

func GetAdminRootPath() string {
	return adminRoot
}
func CreateAdminRootPath() string {
	return adminPrefix + adminRoot
}

func GetAdminFsPath() string {
	return getPath(adminFsPath, "*fsPath")
}
func CreateAdminFsPath(path string) *data.URLBuilder {
	return data.NewURL(adminPrefix + fmt.Sprintf(adminFsPath, path))
}

func GetAdminImgPath() string {
	return getPath(adminImgPath, ":id")
}
func CreateAdminImgPath(imgID uint64) string {
	return adminPrefix + fmt.Sprintf(adminImgPath, imgID)
}
