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

func classifyAspect(w, h int) gridData.Aspect {
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

func assignRolesForLayout(tmp []*gridData.GridImage, layoutWidth int, matrix gridConfig.GridConfig, salt uint64) {
	// 1. base cell count (normal role)
	totalBase := 0
	for _, img := range tmp {
		layout, ok := img.Layouts[layoutWidth]
		if !ok {
			// FIXME
			log.Logger.Error().Int("missing width", layoutWidth).Msg("sh*t happens: no layout")
			continue
		}
		totalBase += layout.Rect.W * layout.Rect.H
	}

	maxCells := totalBase + totalBase/3
	extraCells := maxCells - totalBase

	heroBudget := int(float64(extraCells) * heroShare)
	largeBudget := extraCells - heroBudget

	for _, img := range tmp {
		layout, ok := img.Layouts[layoutWidth]
		if !ok {
			// FIXME
			log.Logger.Error().Msg("sh*t happens: no layout")
		}
		cells := layout.Rect.W * layout.Rect.H
		if heroBudget > 0 {
			sp, _ := matrix.Span(layoutWidth, gridData.RoleHero, img.AspectClass)
			delta := sp.W*sp.H - cells
			if heroBudget-delta >= 0 {
				heroBudget -= delta
				layout.Role = gridData.RoleHero
				layout.Rect = sp.ToRect()
				img.Layouts[layoutWidth] = layout
				continue
			}
		}
		if largeBudget > 0 {
			sp, _ := matrix.Span(layoutWidth, gridData.RoleLarge, img.AspectClass)
			delta := sp.W*sp.H - cells
			if largeBudget-delta >= 0 {
				largeBudget -= delta
				layout.Role = gridData.RoleLarge
				layout.Rect = sp.ToRect()
				img.Layouts[layoutWidth] = layout
				continue
			}
		}
		if largeBudget == 0 && heroBudget == 0 {
			break
		}
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

func BuildGrid(images []dbo.Image, gridConfig gridConfig.GridConfig, salt uint64) []*gridData.GridImage {
	ret := make([]*gridData.GridImage, 0)
	if len(images) == 0 {
		return ret
	}

	// fill table
	// default cell count
	for _, img := range images {
		aspectClass := classifyAspect(int(img.Width), int(img.Height))
		caption := utils.FromStringPtr(img.Title)
		if caption == "" {
			caption = img.Filename
		}
		rating := 0
		if img.Rating != nil {
			rating = int(*img.Rating)
		}
		// TODO multiple width
		layout := BuildForLayout(gridConfig, 12, aspectClass)
		gi := gridData.GridImage{
			ImgId:       *img.ID,
			Caption:     caption,
			Focus:       data.ResolveFocus(img.FocusX, img.FocusY, data.ImageFocusMode(img.FocusMode)),
			Rating:      rating,
			AspectClass: aspectClass,
			Layouts:     map[int]*gridData.Layout{12: layout},
			Aspect:      float32(img.Width) / float32(img.Height),
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

	// TODO: multiple
	//assignRolesForLayout(tmp, 12, placement.DefaultGridMatrix, salt)
	assignRolesForLayouts(tmp, []int{12}, gridConfig)

	for _, img := range ret {
		// TODO: params
		for _, l := range img.Layouts {
			l.ComputeClamp(5.0/4.0, img.Focus, img.Aspect)
		}
	}

	// placement 12
	// TODO for loop

	PlaceTilesSkyline(ret, 12)

	return ret
}

func BuildForLayout(grid gridConfig.GridConfig, width int, aspect gridData.Aspect) *gridData.Layout {
	s, ok := grid.Span(width, gridData.RoleNormal, aspect)
	if !ok {
		log.Logger.Error().Int("width", width).Msg("Layout has no config for this width")
		return nil
	}
	return &gridData.Layout{
		Width: width,
		Rect:  s.ToRect(),
	}
}
