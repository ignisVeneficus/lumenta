package grid

import (
	"sort"

	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"
	"github.com/rs/zerolog/log"

	gridConfig "github.com/ignisVeneficus/lumenta/config/presentation"
	gridData "github.com/ignisVeneficus/lumenta/tpl/grid/data"
)

const (
	heroShare = 0.6
	heroQty   = 2
	largeQty  = 5
)

func mix64(x uint64) uint64 {
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return x
}

func rankKey(img *gridData.GridImage, salt uint64) (int, uint64) {
	h := mix64(img.ImgId ^ salt)
	return img.Rating, h
}

func calculateAspect(w, h int) float32 {
	return float32(w) / float32(h)
}

func ClassifyAspect(w, h int) gridData.Aspect {
	r := calculateAspect(w, h)

	switch {
	case r > 2.6:
		return gridData.AspectPanorama
	case r > 1.4:
		return gridData.AspectLandscape
	case r < 0.8:
		return gridData.AspectTall
	default:
		return gridData.AspectNormal
	}
}

func assignRolesForLayouts(tmp []*gridData.GridImage, layoutWidths []int, matrix gridConfig.GridConfig) {
	heros := heroQty
	larges := largeQty
	for _, img := range tmp {
		role := gridData.RoleNormal
		switch {
		case heros > 0:
			heros--
			role = gridData.RoleHero
		case larges > 0:
			larges--
			role = gridData.RoleLarge
		}

		for _, width := range layoutWidths {
			layout, ok := img.Layouts[width]
			if !ok {
				// FIXME
				log.Logger.Error().Int("missing width", width).Msg("sh*t happens: no layout")
				continue
			}
			layout.Role = role
			span, ok := matrix.Span(width, role, img.AspectClass)
			if ok {
				layout.Rect = span.ToRect()
			}
		}
	}
}

func BuildGrid(images []dbo.Image, gridCfg gridConfig.GridConfig, salt uint64, urlBuilder URLBuilder) []*gridData.GridImage {
	ret := make([]*gridData.GridImage, 0)
	if len(images) == 0 {
		return ret
	}
	grids := []int{}
	for i := range gridCfg {
		grids = append(grids, i)
	}
	// fill table
	// default cell count
	for _, img := range images {
		aspectClass := ClassifyAspect(int(img.Width), int(img.Height))
		caption := utils.FromStringPtr(img.Title)
		if caption == "" {
			caption = img.Filename
		}
		rating := 0
		if img.Rating != nil {
			rating = int(*img.Rating)
		}
		layouts := map[int]*gridData.Layout{}
		for _, i := range grids {
			layout := BuildForLayout(gridCfg, i, aspectClass)
			layouts[i] = layout
		}
		gi := gridData.GridImage{
			ImgId:       *img.ID,
			Caption:     caption,
			Focus:       data.ResolveFocus(img.FocusX, img.FocusY, data.ImageFocusMode(img.FocusMode)),
			Rating:      rating,
			AspectClass: aspectClass,
			Layouts:     layouts,
			Aspect:      float32(img.Width) / float32(img.Height),
			URL:         urlBuilder(*img.ID),
		}
		ret = append(ret, &gi)
	}

	// 2. ranking
	tmp := make([]*gridData.GridImage, len(images))
	copy(tmp, ret)
	sort.Slice(tmp, func(i, j int) bool {
		ri, hi := rankKey(tmp[i], salt)
		rj, hj := rankKey(tmp[j], salt)

		if ri != rj {
			return ri > rj
		}
		return hi < hj
	})

	assignRolesForLayouts(tmp, grids, gridCfg)

	for _, img := range ret {
		// TODO: params
		for _, l := range img.Layouts {
			l.ComputeClamp(5.0/4.0, img.Focus, img.Aspect)
		}
	}

	for _, i := range grids {
		PlaceTilesSkyline(ret, i)
	}

	return ret
}

func BuildForLayout(gridCfg gridConfig.GridConfig, width int, aspect gridData.Aspect) *gridData.Layout {
	s, ok := gridCfg.Span(width, gridData.RoleNormal, aspect)
	if !ok {
		log.Logger.Error().Int("width", width).Msg("Layout has no config for this width")
		return nil
	}
	return &gridData.Layout{
		Width: width,
		Rect:  s.ToRect(),
	}
}

type URLBuilder func(imgID uint64) string
