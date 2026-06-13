package tpl

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/config/presentation"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/server/routes"
	tplData "github.com/ignisVeneficus/lumenta/tpl/data"
	"github.com/ignisVeneficus/lumenta/utils"
)

func CreateImage(c context.Context, cfg config.Config, db *sql.DB, image dbo.Image, acl dbo.ACLContext) (tplData.PageImage, error) {
	logScope, ctx := logging.Enter(c, "tpl/utils/image/create", image.ID, map[string]any{
		"image": image,
	})
	// TODO: use visibility settings

	flatForrest := tplData.NewFlatForest()
	tagsList := tplData.MapToViewNodes(image.Tags, func(t *dbo.Tag) tplData.ViewTreeNode {
		return tplData.ViewTreeNode{
			ID:       uint64(*t.ID),
			ParentID: (*uint64)(t.ParentID),
			Label:    t.Name,
			URL:      template.URL(routes.CreateTagPath(routes.TagID(*t.ID))),
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
	albumIds, err := dao.QueryAlbumsIDByImageID(db, ctx, *image.ID)
	if err != nil {
		logging.ExitErr(logScope, err)
		return tplData.PageImage{}, err
	}
	albums, err := CollectAlbumsByACL(db, ctx, acl, albumIds)

	albumsTree := CreateImageAlbums(albums, func(id dbo.AlbumID) template.URL {
		return template.URL(routes.CreateAlbumPath(routes.AlbumID(id)))
	})

	similars, err := CreateImageSimilars(ctx, db, cfg.Presentation.TagMeaningConfig, image.Tags, acl)

	ret := tplData.PageImage{
		Image:     image,
		SingleMap: singleMap,
		Tags:      *forest,
		Albums:    albumsTree,
		SameTags:  similars,
	}

	ret.Metadata = handleImgMetadata(ctx, cfg, image)
	logging.Exit(logScope, "ok", nil)
	return ret, nil
}
func CollectAlbumsByACL(db *sql.DB, c context.Context, acl dbo.ACLContext, albums []dbo.AlbumID) ([]*dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "tpl/utils/image/albums/collect/acl", nil, nil)
	albumIDs := make(map[dbo.AlbumID]dbo.Album)
	imageAlbums := make([]*dbo.Album, 0)
	for len(albums) > 0 {
		id := albums[0]
		albums = albums[1:]
		if _, ok := albumIDs[id]; ok {
			continue
		}
		album, err := dao.GetAlbumByIDACL(db, ctx, id, acl)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			continue
		case err != nil:
			logging.ExitErr(logScope, err)
			return nil, err
		}
		albumIDs[id] = album
		imageAlbums = append(imageAlbums, &album)
		albums = append(albums, album.AncestorIDs...)
	}
	logging.Exit(logScope, "ok", nil)
	return imageAlbums, nil
}
func CollectAlbums(db *sql.DB, c context.Context, albumsID []dbo.AlbumID) ([]*dbo.Album, error) {
	logScope, ctx := logging.Enter(c, "tpl/utils/image/albums/collect/acl", nil, nil)
	albumIDs := make(map[dbo.AlbumID]dbo.Album)
	imageAlbums := make([]*dbo.Album, 0)
	for len(albumsID) > 0 {
		id := albumsID[0]
		albumsID = albumsID[1:]
		if _, ok := albumIDs[id]; ok {
			continue
		}
		album, err := dao.GetAlbumByID(db, ctx, id)
		switch {
		case errors.Is(err, dao.ErrDataNotFound):
			continue
		case err != nil:
			logging.ExitErr(logScope, err)
			return nil, err
		}
		albumIDs[id] = album
		imageAlbums = append(imageAlbums, &album)
		albumsID = append(albumsID, album.AncestorIDs...)
	}
	logging.Exit(logScope, "ok", nil)
	return imageAlbums, nil

}
func CreateImageAlbums(albums []*dbo.Album, urlGen func(dbo.AlbumID) template.URL) data.Forest[*tplData.ViewTreeNode] {

	albumIDs := make(map[dbo.AlbumID]*dbo.Album)
	for _, album := range albums {
		albumIDs[*album.ID] = album
	}
	flatForrest := tplData.NewFlatForest()

	albumList := tplData.MapToViewNodes(albums, func(a *dbo.Album) tplData.ViewTreeNode {
		return tplData.ViewTreeNode{
			ID:       uint64(*a.ID),
			ParentID: (*uint64)(a.ParentID),
			Label:    a.Name,
			URL:      urlGen(*a.ID),
			Title:    utils.FromStringPtr(a.Description),
		}
	})
	flatForrest.Add(albumList)
	return *flatForrest.Build()
}
func CreateImageSimilars(c context.Context, db *sql.DB, tagMeaningConfig *presentation.TagMeaningConfig, imageTags []*dbo.Tag, acl dbo.ACLContext) ([]tplData.SameTags, error) {
	logScope, ctx := logging.Enter(c, "tpl/utils/image/albums", nil, nil)
	tags, err := dao.QueryTagsByACL(db, ctx, acl)
	if err != nil {
		logging.ExitErr(logScope, err)
		return nil, err
	}
	tagsPoi := dbo.TagsWCountToPointer(tags)
	tagDiscovery := tplData.CreateTagDiscovery(tagMeaningConfig, tagsPoi)
	result := tagDiscovery.GetSame(imageTags)
	logging.Debug(logScope, "after_result", map[string]any{"result": result})
	ret := make([]tplData.SameTags, 0)
	for m, list := range result {
		item := tplData.SameTags{
			Type: string(m),
		}
		for _, t := range list {
			item.Links = append(item.Links, tplData.Link{
				URL:      routes.CreateTagPath(routes.TagID(*t.ID)),
				Label:    t.Name,
				TitleKey: "nav.page.common.browse",
				TitleMap: map[string]interface{}{
					"folder": t.Name,
				},
			})
		}
		ret = append(ret, item)
	}
	logging.Exit(logScope, "ok", nil)
	return ret, nil
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
	ret := tplData.Breadcrumbs{
		tplData.Breadcrumb{
			Link: tplData.Link{
				LabelKey: "nav.page.public.tags.short",
				TitleKey: "nav.page.public.tags.label",
				URL:      routes.CreateTagsRootPath(),
			},
			Type: "page",
		},
	}
	end := len(tags)
	if last {
		end--
	}
	for i := 0; i < end; i++ {
		brc := tplData.Breadcrumb{
			Link: tplData.Link{
				Label:    tags[i].Name,
				TitleKey: "nav.page.common.browse",
				TitleMap: map[string]interface{}{
					"folder": tags[i].Name,
				},
				URL: routes.CreateTagPath(routes.TagID(*tags[i].ID)),
			},
			Type: "tag",
		}
		ret = append(ret, brc)
	}
	if last {
		brc := tplData.Breadcrumb{
			Link: tplData.Link{
				Label: tags[len(tags)-1].Name,
			},
			Type: "tag",
		}
		ret = append(ret, brc)
	}
	return ret

}

func BuildAlbumBreadcumb(database *sql.DB, c context.Context, album dbo.Album, acl dbo.ACLContext, last bool) (tplData.Breadcrumbs, error) {
	ret := tplData.Breadcrumbs{
		tplData.Breadcrumb{
			Link: tplData.Link{
				LabelKey: "nav.page.public.albums.short",
				Title:    "nav.page.public.albums.long",
				URL:      routes.CreateAlbumsRootPath(),
			},
			Type: "albums",
		},
	}
	ids := album.AncestorIDs
	ids = ids[:len(ids)-1]

	for _, id := range ids {
		a, err := dao.GetAlbumByIDACL(database, c, id, acl)
		switch {
		case err == nil:
			br := tplData.Breadcrumb{
				Link: tplData.Link{
					Label: a.Name,
					Title: utils.FromStringPtr(a.Description),
					URL:   routes.CreateAlbumPath(routes.AlbumID(id)),
				},
				Type: "album",
			}
			ret = append(ret, br)
		case !errors.Is(err, dao.ErrDataNotFound):
			return ret, err
		}
	}
	br := tplData.Breadcrumb{
		Link: tplData.Link{
			Label: album.Name,
		},
		Type: "album",
	}
	if !last {
		br.URL = routes.CreateAlbumPath(routes.AlbumID(*album.ID))
		br.Title = utils.FromStringPtr(album.Description)
	}
	ret = append(ret, br)
	return ret, nil

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
func ParseAlbumID(s string) (routes.AlbumID, error) {
	id, err := utils.ParseUint(s)
	return routes.AlbumID(id), err
}
func ParseImageID(s string) (routes.ImageID, error) {
	id, err := utils.ParseUint(s)
	return routes.ImageID(id), err
}
func ParseTagID(s string) (routes.TagID, error) {
	id, err := utils.ParseUint(s)
	return routes.TagID(id), err
}
func ParseSyncRunID(s string) (routes.SyncRunID, error) {
	id, err := utils.ParseUint(s)
	return routes.SyncRunID(id), err
}
func ParseSyncFileID(s string) (routes.SyncFileID, error) {
	id, err := utils.ParseUint(s)
	return routes.SyncFileID(id), err
}
func ParsePaging(s string) (uint64, error) {
	return utils.ParseEmptyUint(s)
}

func UintToString(val *uint64) string {
	if val == nil {
		return ""
	}
	return strconv.FormatUint(*val, 10)
}

func AlbumIDToString(val *dbo.AlbumID) string {
	if val == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*val), 10)
}

func CreateSpacePath(fullpath string) string {
	return strings.ReplaceAll(fullpath, "/", "/\u200B")
}
func L(c *gin.Context) string {
	a := auth.GetAuthContex(c)
	return a.Locale
}

func GetAdminMain() tplData.Breadcrumb {
	return tplData.Breadcrumb{
		Link: tplData.Link{
			LabelKey: "nav.page.admin.home.short",
			URL:      routes.CreateAdminRootPath(),
			TitleKey: "nav.page.admin.home.label",
		},
		Type: "page",
	}
}
