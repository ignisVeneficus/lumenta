package derivative

import (
	"context"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	derivativeConfig "github.com/ignisVeneficus/lumenta/config/derivative"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/logging"
)

func applyTask(ctx context.Context, t Task, img image.Image, focus data.Focus) error {
	logg := logging.Enter(ctx, "derivative.generic.task", map[string]any{"task": t})

	targetW := t.Mode.MaxWidth
	targetH := t.Mode.MaxHeight

	var err error
	switch t.Mode.Mode {
	case derivativeConfig.DerivativeSizeResize:
		img, err = resizeFit(img, targetW, targetH)
		if err != nil {
			logging.ExitErr(logg, err)
			return err
		}

	case derivativeConfig.DerivativeSizeCrop:
		img = resizeCrop(img, targetW, targetH, focus)

	default:
		err = fmt.Errorf("unknown resize mode: %s", t.Mode.Mode)
		logging.ExitErr(logg, err)
		return err
	}
	if err := writeImage(ctx, img, t.TargetPath); err != nil {
		logging.ExitErr(logg, err)
	}
	logging.Exit(logg, "ok", nil)
	return nil
}

func GenerateDerivativeStep(j *Job) error {
	logg := logging.Enter(j.Ctx, "derivative.generic.step", map[string]any{"job": j})

	img, err := imaging.Open(j.SourcePath, imaging.AutoOrientation(false))
	if err != nil {
		logging.ExitErrParams(logg, err, map[string]any{"path": j.SourcePath, "step": "open"})
		return err
	}

	img = applyRotation(img, j.ImageParams.Rotation)

	j.ImageParams.Focus.Rotate(j.ImageParams.Rotation)

	for _, t := range j.Tasks {
		err := applyTask(j.Ctx, t, img, j.ImageParams.Focus)
		if err != nil {
			logging.ErrorContinue(logg, err, map[string]any{"task": t.Mode.Name})
		}
	}

	logging.Exit(logg, "ok", nil)
	return nil
}

func applyRotation(img image.Image, deg int16) image.Image {
	switch ((deg % 360) + 360) % 360 {
	case 90:
		// EXIF: Rotate 90 CW → imaging: Rotate270
		return imaging.Rotate270(img)
	case 180:
		return imaging.Rotate180(img)
	case 270:
		// EXIF: Rotate 270 CW → imaging: Rotate90
		return imaging.Rotate90(img)
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

func cropToTarget(img image.Image, targetW, targetH int, focus data.Focus) image.Image {

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	cx := int(math.Round(float64(w) * float64(focus.FocusX)))
	cy := int(math.Round(float64(h) * float64(focus.FocusY)))

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

func resizeCrop(img image.Image, targetW, targetH int, focus data.Focus) image.Image {

	// 1. scale-to-cover
	img = resizeCover(img, targetW, targetH)
	// 2. crop to exact size using focus
	return cropToTarget(img, targetW, targetH, focus)
}

func writeImage(ctx context.Context, img image.Image, path string) error {
	logg := logging.Enter(ctx, "derivative.generic.step.write", map[string]any{"path": path})

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
