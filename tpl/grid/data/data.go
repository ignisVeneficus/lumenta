package grid

import (
	"github.com/ignisVeneficus/lumenta/data"
)

type Aspect string

const (
	AspectNormal    Aspect = "normal"
	AspectLandscape Aspect = "landscape"
	AspectPanorama  Aspect = "panorama"
	AspectTall      Aspect = "tall"
)

func (a Aspect) Valid() bool {
	switch a {
	case AspectNormal, AspectLandscape, AspectPanorama, AspectTall:
		return true
	default:
		return false
	}
}

type Role string

const (
	RoleNormal Role = "normal"
	RoleLarge  Role = "large"
	RoleHero   Role = "hero"
)

func (r Role) Valid() bool {
	switch r {
	case RoleNormal, RoleLarge, RoleHero:
		return true
	default:
		return false
	}
}

type Span struct {
	W int `yaml:"w"`
	H int `yaml:"h"`
}

type Clamp struct {
	MinX float32
	MaxX float32
	MinY float32
	MaxY float32
}

type Layout struct {
	Width int
	Clamp Clamp
	Rect  *Rect
	Role  Role
}

func (l *Layout) ComputeClamp(cellAspect float32, focus data.Focus, imageAspect float32) {
	boxAspect := (float32(l.Rect.W) * cellAspect) / float32(l.Rect.H)

	switch {
	case imageAspect > boxAspect:
		// Image is wider -> horizontal crop
		// Height fits exactly
		scale := boxAspect / imageAspect
		// visible fraction of image width
		visibleX := scale / 2
		l.Clamp = Clamp{
			MinX: visibleX,
			MaxX: 1 - visibleX,
			MinY: 0.0,
			MaxY: 1.0,
		}
	case imageAspect < boxAspect:
		// Image is taller -> vertical crop
		// Width fits exactly
		scale := imageAspect / boxAspect
		// visible fraction of image height
		visibleY := scale / 2
		l.Clamp = Clamp{
			MinX: 0.0,
			MaxX: 1.0,
			MinY: visibleY,
			MaxY: 1 - visibleY,
		}
	default:
		l.Clamp = Clamp{
			MinX: 0.5,
			MaxX: 0.5,
			MinY: 0.5,
			MaxY: 0.5,
		}
	}
}

type Rect struct {
	X int
	Y int
	W int
	H int
}

func (s Span) ToRect() *Rect {
	return &Rect{
		W: s.W,
		H: s.H,
	}
}

type GridImage struct {
	ImgId       uint64
	Caption     string
	Focus       data.Focus
	AspectClass Aspect
	Aspect      float32
	Rating      int
	Layouts     map[int]*Layout
	URL         string
}

func (gi *GridImage) ComputeClamp(cellAspect float32) {
	for _, l := range gi.Layouts {
		l.ComputeClamp(cellAspect, gi.Focus, gi.Aspect)
	}
}
func (gi *GridImage) GetRec(width int) *Rect {
	l, ok := gi.Layouts[width]
	if !ok {
		return nil
	}
	return l.Rect
}
