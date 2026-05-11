package tpl

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/internal/i18n"
	"github.com/ignisVeneficus/lumenta/server/routes"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/utils"
)

func CreateImage(c context.Context, cfg config.Config, image dbo.Image) tplData.PageImage {
	logScope, ctx := logging.Enter(c, "tpl/utils/image/create", image.ID, map[string]any{
		"image": image,
	})
	// TODO: use visibility settings

	flatForrest := tplData.NewFlatForrest()
	tagsList := tplData.MapToViewNodes(image.Tags, func(t *dbo.Tag) tplData.ViewTreeNode {
		return tplData.ViewTreeNode{
			ID:       *t.ID,
			ParentID: t.ParentID,
			Label:    t.Name,
			URL:      template.URL(routes.CreateTagPath(*t.ID)),
		}
	})
	flatForrest.Add(tagsList)
	forest := flatForrest.Build()

	tplData.SetTagsMeaning(forest, cfg.Presentation.TagMeaningConfig)

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
		Tags:      *forest,
	}

	ret.Metadata = handleImgMetadata(ctx, cfg, image)
	logging.Exit(logScope, "ok", nil)
	return ret
}

/*
	func appendMetadata(list []tplData.MetadataValue, label string, key string, m data.Metadata) []tplData.MetadataValue {
		mtv, ok := m[key]
		if ok {
			list = append(list, createMetadata(label, mtv))
		}
		return list
	}
*/
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

func BuildTagBreadcumb(database *sql.DB, c context.Context, tag dbo.Tag, last bool) (tplData.Breadcrumbs, error) {
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
	logScope, _ := logging.Enter(ctx, "tpl/utils/metadata/unmarshal", img.ID, map[string]any{
		"json":     img.ExifJSON,
		"image_id": img.ID,
	})
	m := data.Metadata{}
	err := json.Unmarshal([]byte(img.ExifJSON), &m)
	return m, logging.Return(logScope, err)
}

func ParseID(s string) (uint64, error) {
	if s == "" {
		return 0, tplData.ErrMissingMandatoryValue
	}
	return strconv.ParseUint(s, 10, 64)
}
func ParsePaging(s string) (uint64, error) {
	if s == "" {
		return 1, nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	if v == 0 {
		return 0, tplData.ErrInvalidValue
	}
	return v, nil
}

func UintToString(val *uint64) string {
	if val == nil {
		return ""
	}
	return strconv.FormatUint(*val, 10)
}

func CreateSpacePath(fullpath string) string {
	return strings.ReplaceAll(fullpath, "/", "/\u200B")
}
func L(c *gin.Context) string {
	a := auth.GetAuthContex(c)
	return a.Locale
}

func GetAdminMain(lang string, i18n *i18n.Service) tplData.Breadcrumb {
	return tplData.Breadcrumb{
		Label: i18n.T(lang, "nav.page.admin.home.short", nil),
		Link:  template.URL(routes.CreateAdminRootPath()),
		Type:  "page",
		Title: i18n.T(lang, "nav.page.admin.home.label", nil),
	}
}
