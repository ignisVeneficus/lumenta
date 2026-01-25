package derivates

import (
	"context"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/logging"
)

func GenerateDerivateStep(j *Job) error {
	logg := logging.Enter(j.Ctx, "derivates.generic.step", map[string]any{"job": j})

	targetW := j.Mode.MaxWidth
	targetH := j.Mode.MaxHeight

	img, err := imaging.Open(j.SourcePath, imaging.AutoOrientation(false))
	if err != nil {
		logging.ExitErrParams(logg, err, map[string]any{"path": j.SourcePath, "step": "open"})
		return err
	}

	img = applyRotation(img, j.ImageParams.Rotation)

	switch j.Mode.Mode {
	case config.DerivateSizeResize:
		img, err = resizeFit(img, targetW, targetH)
		if err != nil {
			logging.ExitErr(logg, err)
			return err
		}

	case config.DerivateSizeCrop:
		focusX, focusY := resolveFocus(j.ImageParams.FocusX, j.ImageParams.FocusY, j.ImageParams.FocusMode)
		focusX, focusY = rotateFocus(focusX, focusY, j.ImageParams.Rotation)
		img = resizeCrop(img, targetW, targetH, focusX, focusY)

	default:
		err = fmt.Errorf("unknown resize mode: %s", j.Mode.Mode)
		logging.ExitErr(logg, err)
		return err
	}
	if err := writeImage(j.Ctx, img, j.TargetPath); err != nil {
		logging.ExitErr(logg, err)
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func applyRotation(img image.Image, deg int16) image.Image {
	switch ((deg % 360) + 360) % 360 {
	case 90:
		return imaging.Rotate90(img)
	case 180:
		return imaging.Rotate180(img)
	case 270:
		return imaging.Rotate270(img)
	default:
		return img
	}
}

func resizeFit(img image.Image, maxW, maxH int) (image.Image, error) {

	b := img.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()

	if srcW <= 0 || srcH <= 0 {
		return nil, fmt.Errorf("invalid image, 0px")
	}

	if maxW <= 0 {
		scale := float64(maxH) / float64(srcH)
		maxW = int(math.Round(float64(srcW) * scale))
		if maxW < 1 {
			maxH = 1
		}
	}
	if maxH <= 0 {
		scale := float64(maxW) / float64(srcW)
		maxH = int(math.Round(float64(srcH) * scale))
		if maxH < 1 {
			maxH = 1
		}
	}
	return imaging.Fit(
		img,
		maxW,
		maxH,
		imaging.Lanczos,
	), nil
}
func resolveFocus(fpx, fpy *float32, mode ImageFocusMode) (fx, fy float32) {

	if fpx != nil && fpy != nil {
		return *fpx, *fpy
	}
	switch mode {
	case ImageFocusModeTop:
		return 0.5, 0.0
	case ImageFocusModeBottom:
		return 0.5, 1.0
	case ImageFocusModeLeft:
		return 0.0, 0.5
	case ImageFocusModeRight:
		return 1.0, 0.5
	case ImageFocusModeCenter:
		return 0.5, 0.5
	case ImageFocusModeAuto:
		return 0.5, 0.5
	default:
		return 0.5, 0.5
	}
}

/*
	func autoFocus(img image.Image) (fx, fy float64) {
		// edge density
		// face detection
		// saliency map
		return 0.5, 0.5
	}
*/
func rotateFocus(fx, fy float32, rotation int16) (nfx, nfy float32) {
	switch ((rotation % 360) + 360) % 360 {
	case 90:
		return 1.0 - fy, fx
	case 180:
		return 1.0 - fx, 1.0 - fy
	case 270:
		return fy, 1.0 - fx
	default:
		return fx, fy
	}
}

func resizeCover(img image.Image, targetW, targetH int) image.Image {

	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()

	scaleW := float64(targetW) / float64(srcW)
	scaleH := float64(targetH) / float64(srcH)

	scale := math.Max(scaleW, scaleH)

	newW := int(math.Ceil(float64(srcW) * scale))
	newH := int(math.Ceil(float64(srcH) * scale))

	return imaging.Resize(img, newW, newH, imaging.Lanczos)
}

func cropToTarget(img image.Image, targetW, targetH int, fx, fy float64) image.Image {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	cx := int(math.Round(float64(w) * fx))
	cy := int(math.Round(float64(h) * fy))

	x0 := cx - targetW/2
	y0 := cy - targetH/2

	// clamp
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x0+targetW > w {
		x0 = w - targetW
	}
	if y0+targetH > h {
		y0 = h - targetH
	}

	rect := image.Rect(x0, y0, x0+targetW, y0+targetH)
	return imaging.Crop(img, rect)
}

func resizeCrop(img image.Image, targetW, targetH int, fx, fy float32) image.Image {

	// 1. scale-to-cover
	img = resizeCover(img, targetW, targetH)
	// 2. crop to exact size using focus
	return cropToTarget(img, targetW, targetH, float64(fx), float64(fy))
}

func writeImage(ctx context.Context, img image.Image, path string) error {
	logg := logging.Enter(ctx, "derivates.generic.step.write", map[string]any{"path": path})

	tmp := path + ".tmp"

	dir := filepath.Dir(tmp)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logging.ExitErrParams(logg, err, map[string]any{"step": "create dirs"})
		return err
	}
	out, err := os.Create(tmp)
	if err != nil {
		logging.ExitErrParams(logg, err, map[string]any{"step": "create file"})
		return err
	}
	if err := imaging.Encode(out, img, imaging.JPEG, imaging.JPEGQuality(85)); err != nil {
		out.Close()
		os.Remove(tmp)
		logging.ExitErrParams(logg, err, map[string]any{"step": "write"})
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		logging.ExitErrParams(logg, err, map[string]any{"step": "sync"})
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		logging.ExitErrParams(logg, err, map[string]any{"step": "close"})
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		logging.ExitErrParams(logg, err, map[string]any{"step": "rename"})
		return err
	}
	return nil
}
