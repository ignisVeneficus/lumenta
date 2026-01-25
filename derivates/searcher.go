package derivates

import (
	"context"

	"github.com/ignisVeneficus/lumenta/auth"
	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/db"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/logging"
	"github.com/ignisVeneficus/lumenta/utils"
)

func createDBOACL(acl auth.ACLContext) dao.ACLContext {
	return dao.ACLContext{
		ViewerUserID: acl.UserID,
		Role:         string(acl.Role),
	}
}

func GetDerivatesPathWithACL(ctx context.Context, acl auth.ACLContext, imageId uint64, cfg config.DerivativeConfig, roots config.MediaConfig) (string, error) {
	logg := logging.Enter(ctx, "middleware.getDerivates", map[string]any{"image_id": imageId, "derivate": cfg.Name})
	db := db.GetDatabase()
	image, err := dao.GetImageByIdACL(db, context.Background(), imageId, createDBOACL(acl))
	if err != nil {
		logging.ExitErr(logg, err)
		return "", err
	}
	outPath := utils.ConcatGlobalDerivatedPath(roots.Derivatives, image.Path, image.Filename, cfg.Postfix, image.Ext)

	ok, err := utils.FileExists(outPath)
	if ok {
		logging.Exit(logg, "found", map[string]any{"path": outPath})
		return outPath, nil
	}

	inPath := utils.ConcatGlobalPath(roots.Originals, image.Path, image.Filename, image.Ext)
	rot := int16(0)
	if image.Rotation != nil {
		rot = *image.Rotation
	}
	imageParams := ImageParams{
		FocusMode: ImageFocusMode(image.FocusMode),
		FocusX:    image.FocusX,
		FocusY:    image.FocusY,
		Rotation:  rot,
	}
	job := Job{
		Key:         Key(outPath),
		Image:       *image.ID,
		Mode:        cfg,
		SourcePath:  inPath,
		TargetPath:  outPath,
		ImageParams: imageParams,
		Ctx:         ctx,
	}
	service := Get()
	ok, err = service.Submit(job)
	if err != nil {
		logging.ExitErr(logg, err)
	}
	logging.Exit(logg, "create", map[string]any{"path": outPath})
	return outPath, nil
}
