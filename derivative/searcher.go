package derivative

import (
	"context"
	"errors"
	"fmt"

	"github.com/ignisVeneficus/logging"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	derivativeConfig "github.com/ignisVeneficus/lumenta/config/derivative"
	fsConfig "github.com/ignisVeneficus/lumenta/config/filesystem"
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/utils"
)

func GetDerivativesPathWithACL(c context.Context, acl authData.ACLContext, imageID uint64, cfg derivativeConfig.DerivativeConfig, roots fsConfig.FilesystemConfig) (string, error) {
	logScope, ctx := logging.Enter(c, "middleware/derivatives", imageID, map[string]any{"image_id": imageID, "derivative": cfg.Name})
	db := db.GetDatabase()
	image, err := dao.GetImageByIdACL(db, c, dbo.ImageID(imageID), acl.ACLContext)
	if err != nil {
		logging.ExitErr(logScope, err)
		return "", err
	}
	outPath := utils.ConcatGlobalDerivativePath(roots.Derivatives, image.Root, image.Path, image.Filename, cfg.Postfix, "jpg")

	ok, err := utils.FileExists(outPath)
	if ok {
		logging.Exit(logScope, "found", map[string]any{"path": outPath})
		return outPath, nil
	}
	imgRoot, ok := roots.Originals[image.Root]
	if !ok {
		err := fmt.Errorf("root not defined: %s", image.Root)
		logging.ExitErr(logScope, err)
		return "", err
	}
	inPath := utils.ConcatGlobalPath(imgRoot.Root, image.Path, image.Filename, image.Ext)
	rot := int16(0)
	if image.Rotation != nil {
		rot = *image.Rotation
	}
	imageParams := ImageParams{
		Focus:    data.ResolveFocus(image.FocusX, image.FocusY, data.ImageFocusMode(image.FocusMode)),
		Rotation: rot,
	}
	job := Job{
		Key:        Key(outPath),
		Image:      uint64(*image.ID),
		SourcePath: inPath,
		Tasks: []Task{
			Task{
				Mode:       cfg,
				TargetPath: outPath,
			},
		},
		ImageParams: imageParams,
		Ctx:         logging.Detach(ctx),
	}
	service := Get()
	ok, err = service.Submit(job)
	if err != nil {
		if !errors.Is(err, ErrDuplicate) {
			logging.ExitErr(logScope, err)
			return outPath, err
		}
	}
	logging.Exit(logScope, "create", map[string]any{"path": outPath})
	return outPath, nil
}
