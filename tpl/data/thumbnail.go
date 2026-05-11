package data

import (
	"context"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dbo"
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
	logScope, _ := logging.Enter(c, "tpl/image/thumbnail", image.ID, nil)
	var err error
	// starting point for the query for before/after queries
	thumbnailAfterQty := 0
	thumbnailAfterStart := 0
	// qty for the before after row: display + 1
	// if query return this qty => there are more pages on that side
	thumbnailBeforeQty := 0
	thumbnailBeforeStart := 0
	urlBefore := ""
	urlAfter := ""
	// image of one side the displayed if page is 0 (displayed in middle)
	oneSide := (ThumbnailPerPage - 1) / 2
	var img *uint64 = nil
	if page == 0 {
		// diplayed in middle: both side have same amout images
		// starting point are 0
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
	logging.Trace(logScope, "after paging", map[string]any{
		"thumbnailAfterQty":    thumbnailAfterQty,
		"thumbnailAfterStart":  thumbnailAfterStart,
		"thumbnailBeforeQty":   thumbnailBeforeQty,
		"thumbnailBeforeStart": thumbnailBeforeStart,
		"window":               ThumbnailPerPage,
		"oneSide":              oneSide,
		"page":                 page,
	})

	after := []dbo.ImageTitle{}
	if thumbnailAfterQty > 0 {
		after, err = queryNext(c, image, thumbnailAfterStart, thumbnailAfterQty)

		if err != nil {
			return Thumbnails{}, err
		}
		// we got all the images => next page
		if len(after) == thumbnailAfterQty {
			urlAfter = pagingUrlGenerator(*image.ID, page+1)
		}
	} else if page < 0 {
		// always have next page
		urlAfter = pagingUrlGenerator(*image.ID, page+1)
	}
	before := []dbo.ImageTitle{}
	if thumbnailBeforeQty > 0 {
		before, err = queryPrev(c, image, thumbnailBeforeStart, thumbnailBeforeQty)
		if err != nil {
			return Thumbnails{}, err
		}
		// we got all the images => prev page
		if len(before) == thumbnailBeforeQty {
			urlBefore = pagingUrlGenerator(*image.ID, page-1)
		}
	} else if page > 0 {
		// always have prev image
		urlBefore = pagingUrlGenerator(*image.ID, page-1)
	}
	logging.Exit(logScope, "ok", nil)
	return CreateThumbnails(before, img, after, imageUrlGenerator, ThumbnailPerPage, urlBefore, urlAfter), nil
}
