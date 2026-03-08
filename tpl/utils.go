package tpl

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/gin-gonic/gin"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/server/routes"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/utils"
)

func CreateDBOACL(acl authData.ACLContext) dao.ACLContext {
	return dao.ACLContext{
		ViewerUserID: acl.UserID,
		Role:         string(acl.Role),
	}
}

func CreateImage(ctx context.Context, cfg config.Config, image dbo.Image) tplData.PageImage {
	// TODO: use visibility settings

	var singleMap *tplData.SingleMap = nil

	if image.Latitude != nil && image.Longitude != nil {
		singleMap = &tplData.SingleMap{
			Lat:  *image.Latitude,
			Long: *image.Longitude,
		}
	}
	ret := tplData.PageImage{
		Image:     image,
		SingleMap: singleMap,
		Tags:      image.Tags,
	}

	ret.Metadata = handleImgMetadata(ctx, cfg, image)
	return ret
}
func appendMetadata(list []tplData.MetadataValue, label string, key string, m data.Metadata) []tplData.MetadataValue {
	mtv, ok := m[key]
	if ok {
		list = append(list, createMetadata(label, mtv))
	}
	return list
}
func createMetadata(label string, m data.MetadataValue) tplData.MetadataValue {
	value, _ := m.AsString()
	if value != "" && m.Unit != "" {
		value += m.Unit
	}
	return tplData.MetadataValue{
		Label: label,
		Value: value,
	}
}

func handleImgMetadata(ctx context.Context, cfg config.Config, image dbo.Image) tplData.Metadata {
	// TODO: use visibility settings
	imageMetadata, err := GetImageMetadata(ctx, image)
	if err != nil {
		return tplData.Metadata{}
	}
	ret := tplData.Metadata{
		Title:       utils.FromStringPtr(imageMetadata.GetTitle()),
		Description: utils.FromStringPtr(imageMetadata.GetCaption()),
	}
	delete(imageMetadata, data.MetaTitle)
	delete(imageMetadata, data.MetaCaption)

	blocks := make([]tplData.MetadataBlock, 0)
	photo := tplData.MetadataBlock{
		Label: "Capture",
	}
	photoList := make([]string, 0)
	photoList = addListIfNotEmpty(photoList, imageMetadata, data.MetaCamera)
	photoList = addListIfNotEmpty(photoList, imageMetadata, data.MetaLens)
	params := make([]string, 0)
	params = addListIfNotEmpty(params, imageMetadata, data.MetaFocalLength)
	params = addListIfNotEmpty(params, imageMetadata, data.MetaAperture)
	params = addListIfNotEmpty(params, imageMetadata, data.MetaExposureTime)
	params = addListIfNotEmpty(params, imageMetadata, data.MetaISO)
	paramstr := strings.Join(params, " · ")
	if paramstr != "" {
		photoList = append(photoList, paramstr)
	}
	photo.Data = photoList
	blocks = append(blocks, photo)

	takenAt := tplData.MetadataBlock{
		Label: "Taken at",
	}
	takenAtList := make([]string, 0, 1)
	takenAtList = addListIfNotEmpty(takenAtList, imageMetadata, data.MetaTakenAt)
	takenAt.Data = takenAtList
	blocks = append(blocks, takenAt)

	ret.Blocks = blocks
	// TODO: remaining metada with label system from the presentationConfig

	return ret

}
func addListIfNotEmpty(list []string, data data.Metadata, key string) []string {
	mvalue, ok := data[key]
	if !ok {
		return list
	}
	delete(data, key)
	value, ok := mvalue.AsString()
	if !ok {
		return list
	}
	if value != "" {
		if mvalue.Unit != "" {
			value += " " + mvalue.Unit
		}
		list = append(list, value)
	}
	return list
}

func BuildTagBreadcumb(database *sql.DB, c *gin.Context, tag dbo.Tag, last bool) (tplData.Breadcrumbs, error) {
	path := []dbo.Tag{tag}
	var err error
	for tag.ParentID != nil {
		tag, err = dao.GetTagByID(database, c, *tag.ParentID)
		if err != nil {
			return tplData.Breadcrumbs{}, err
		}
		path = append([]dbo.Tag{tag}, path...)
	}
	return createTagsBreadcrumbs(path, last), nil

}

func createTagsBreadcrumbs(tags []dbo.Tag, last bool) tplData.Breadcrumbs {
	res := tplData.Breadcrumbs{
		tplData.Breadcrumb{
			Label: "Tags",
			Link:  template.URL(routes.CreateTagsRootPath()),
			Type:  "page",
			Title: "View all tags",
		},
	}
	end := len(tags)
	if last {
		end--
	}
	for i := 0; i < end; i++ {
		brc := tplData.Breadcrumb{
			Label: tags[i].Name,
			Link:  template.URL(routes.CreateTagPath(*tags[i].ID)),
			Type:  "tag",
			Title: fmt.Sprintf("View: %s", tags[i].Name),
		}
		res = append(res, brc)
	}
	if last {
		brc := tplData.Breadcrumb{
			Label: tags[len(tags)-1].Name,
			Type:  "tag",
		}
		res = append(res, brc)
	}
	return res

}
func GetImageMetadata(ctx context.Context, img dbo.Image) (data.Metadata, error) {
	m := data.Metadata{}
	err := json.Unmarshal([]byte(img.ExifJSON), &m)
	if err != nil {
		logging.Error(ctx, err, "tpl.getMetadata", "unmashal", "error", "", map[string]any{
			"image_id": img.ID,
		})
	}
	return m, err
}
