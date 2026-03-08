package data

import (
	"context"

	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/logging"
)

func CreateThumbnail(img dbo.ImageTitle, generator func(uint64) *string) *Thumbnail {
	return &Thumbnail{
		ImageId: img.ID,
		Url:     generator(img.ID),
		Title:   img.Title,
	}
}

func CreateThumbnails(before []dbo.ImageTitle, selected *uint64, after []dbo.ImageTitle, generator func(uint64) *string, window int, beforeURL string, afterURL string) Thumbnails {
	thumbs := make([]*Thumbnail, window)
	windowSide := (window - 1) / 2
	if selected != nil {
		thumbs[windowSide] = &Thumbnail{
			ImageId: *selected,
		}
		for i := range windowSide {
			if len(after) <= i {
				break
			}
			img := after[i]
			thumbs[windowSide+i+1] = CreateThumbnail(img, generator)
		}
		for i := range windowSide {
			if len(before) <= i {
				break
			}
			img := before[i]
			thumbs[windowSide-i-1] = CreateThumbnail(img, generator)
		}
	} else {
		if len(before) > len(after) {
			for i := range window {
				if len(before) <= i {
					break
				}
				img := before[i]
				thumbs[window-i-1] = CreateThumbnail(img, generator)
			}
		} else {
			for i := range window {
				if len(after) <= i {
					break
				}
				img := after[i]
				thumbs[i] = CreateThumbnail(img, generator)
			}
		}
	}
	return Thumbnails{
		Images:    thumbs,
		BeforeUrl: beforeURL,
		AfterUrl:  afterURL,
	}
}

type Thumbnails struct {
	BeforeUrl string
	AfterUrl  string
	Images    []*Thumbnail
}

type Thumbnail struct {
	ImageId uint64
	Url     *string
	Title   string
}

func GenerateThumbnail(c context.Context,
	queryPrev func(context.Context, dbo.Image, int, int) ([]dbo.ImageTitle, error),
	queryNext func(context.Context, dbo.Image, int, int) ([]dbo.ImageTitle, error),
	image dbo.Image,
	page int,
	imageUrlGenerator func(uint64) *string,
	pagingUrlGenerator func(uint64, int) string,
) (Thumbnails, error) {
	logg := logging.Enter(c, "tpl.image.thumbnail", nil)
	var err error
	thumbnailAfterQty := 0
	thumbnailAfterStart := 0
	thumbnailBeforeQty := 0
	thumbnailBeforeStart := 0
	urlBefore := ""
	urlAfter := ""
	oneSide := (ThumbnailPerPage - 1) / 2
	var img *uint64 = nil
	if page == 0 {
		thumbnailBeforeQty = oneSide + 1
		thumbnailAfterQty = oneSide + 1
		img = image.ID
	}
	if page > 0 {
		thumbnailAfterQty = ThumbnailPerPage + 1
		thumbnailAfterStart = (page-1)*ThumbnailPerPage + oneSide
	}
	if page < 0 {
		thumbnailBeforeQty = ThumbnailPerPage + 1
		thumbnailBeforeStart = ((-1*page)-1)*ThumbnailPerPage + oneSide
	}
	logging.Inside(logg, map[string]any{
		"thumbnailAfterQty":    thumbnailAfterQty,
		"thumbnailAfterStart":  thumbnailAfterStart,
		"thumbnailBeforeQty":   thumbnailBeforeQty,
		"thumbnailBeforeStart": thumbnailBeforeStart,
		"window":               ThumbnailPerPage,
		"oneSide":              oneSide,
		"page":                 page,
	}, "after paging")

	after := []dbo.ImageTitle{}
	if thumbnailAfterQty > 0 {
		after, err = queryNext(c, image, thumbnailAfterStart, thumbnailAfterQty)
		// dao.QueryImageIDByTagACLNext(database, c, uint64(tagId), *image.ID, image.TakenAt, image.Filename, dboAcl, uint64(thumbnailAfterStart), uint64(thumbnailAfterQty))
		if err != nil {
			return Thumbnails{}, err
		}
		if len(after) == oneSide+1 {
			urlAfter = pagingUrlGenerator(*image.ID, page+1)
			//routes.BuildTagImagePath(tagId, *image.ID).WithIntQuery(imagePageName, page+1).String()
		}
	} else if page < 0 {
		urlAfter = pagingUrlGenerator(*image.ID, page+1)
	}
	before := []dbo.ImageTitle{}
	if thumbnailBeforeQty > 0 {
		before, err = queryPrev(c, image, thumbnailBeforeStart, thumbnailBeforeQty)
		//dao.QueryImageIDByTagACLPrev(database, c, uint64(tagId), *image.ID, image.TakenAt, image.Filename, dboAcl, uint64(thumbnailBeforeStart), uint64(thumbnailBeforeQty))
		if err != nil {
			return Thumbnails{}, err
		}
		if len(before) == oneSide+1 {
			urlBefore = pagingUrlGenerator(*image.ID, page-1)
			//routes.BuildTagImagePath(tagId, *image.ID).WithIntQuery(imagePageName, page-1).String()
		}
	} else if page > 0 {
		urlBefore = pagingUrlGenerator(*image.ID, page-1)
	}

	return CreateThumbnails(before, img, after, imageUrlGenerator, ThumbnailPerPage, urlBefore, urlAfter), nil
}
