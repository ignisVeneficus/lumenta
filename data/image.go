package data

type ImageFocusMode string

const (
	ImageFocusModeAuto   ImageFocusMode = "auto"
	ImageFocusModeManual ImageFocusMode = "manual"
	ImageFocusModeCenter ImageFocusMode = "center"
	ImageFocusModeTop    ImageFocusMode = "top"
	ImageFocusModeBottom ImageFocusMode = "bottom"
	ImageFocusModeLeft   ImageFocusMode = "left"
	ImageFocusModeRight  ImageFocusMode = "right"
)

type Focus struct {
	FocusMode ImageFocusMode
	FocusX    float32
	FocusY    float32
}

func ResolveFocus(fpx, fpy *float32, mode ImageFocusMode) Focus {
	ret := Focus{
		FocusMode: mode,
		FocusX:    0.5,
		FocusY:    0.5,
	}
	switch mode {
	case ImageFocusModeTop:
		ret.FocusX = 0.5
		ret.FocusY = 0.0
	case ImageFocusModeBottom:
		ret.FocusX = 0.5
		ret.FocusY = 1.0
	case ImageFocusModeLeft:
		ret.FocusX = 0.0
		ret.FocusY = 0.5
	case ImageFocusModeRight:
		ret.FocusX = 1.0
		ret.FocusY = 0.5
	case ImageFocusModeCenter:
		ret.FocusX = 0.5
		ret.FocusY = 0.5
	case ImageFocusModeAuto:
		//TODO: auto focus: based on faces
		ret.FocusX = 0.5
		ret.FocusY = 0.5
	case ImageFocusModeManual:
		if fpx != nil && fpy != nil {
			ret.FocusX = *fpx
			ret.FocusY = *fpy
		}
	}
	return ret
}
func (f *Focus) Rotate(rotation int16) {
	fx := f.FocusX
	fy := f.FocusY
	switch ((rotation % 360) + 360) % 360 {
	case 90:
		f.FocusX = 1.0 - fy
		f.FocusY = fx
	case 180:
		f.FocusX = 1.0 - fx
		f.FocusY = 1.0 - fy
	case 270:
		f.FocusX = fy
		f.FocusY = 1.0 - fx
	}
}
