package pipeline

import (
	"encoding/json"
	"time"

	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
	"github.com/ignisVeneficus/lumenta/mapper"
	"github.com/rs/zerolog/log"
)

func UpdateImageMetadata(i *dbo.Image, metadata data.Metadata) error {
	i.Aperture = metadata.GetAperture()
	i.Camera = metadata.GetMakerCamera()
	i.TakenAt = metadata.GetTakenAt()
	i.FocalLength = metadata.GetFocalLength()
	i.Exposure = metadata.GetExposure()
	i.ISO = metadata.GetIso()
	i.Latitude = metadata.GetLatitude()
	i.Longitude = metadata.GetLongitude()
	i.Lens = metadata.GetLens()
	i.Rating = metadata.GetRating()
	i.Width = metadata.GetWidth()
	i.Height = metadata.GetHeight()
	i.Rotation = metadata.GetRotation()
	i.Caption = metadata.GetCaption()
	i.Title = metadata.GetTitle()
	tags := metadata.GetTags()
	JSONMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	i.ExifJSON = JSONMetadata
	i.Tags = dbo.TagsTree{}
	for _, t := range tags {
		tname := mapper.SplitTagPath(t)
		current := &dbo.Tag{Name: tname[len(tname)-1], Source: "digikam"}
		current.Children = nil
		for j := len(tname) - 2; j >= 0; j-- {
			parent := &dbo.Tag{Name: tname[j], Source: "digikam"}
			parent.Children = dbo.TagsTree{current}
			current = parent
		}
		i.Tags = append(i.Tags, current)
	}

	return nil
}

func getDBOImageFromJob(job WorkItem, syncID uint64) {
	job.DBImage.Root = job.RootName
	job.DBImage.Path = job.Path
	job.DBImage.Filename = job.Filename
	job.DBImage.FileSize = uint64(job.Info.Size())
	job.DBImage.MTime = job.Info.ModTime().UTC().Truncate(time.Second)
	job.DBImage.Ext = job.Ext
	job.DBImage.FileHash = job.FileHash
	job.DBImage.MetaHash = job.FileMetadataHash
	job.DBImage.LastSeenSync = &syncID
	if job.Panorama {
		job.DBImage.Panorama = 1
	} else {
		job.DBImage.Panorama = 0
	}
	if job.ACLLevel != nil {
		job.DBImage.ACLLevel = *job.ACLLevel
	}
	job.DBImage.ACLUserID = job.ACLUser
	UpdateImageMetadata(job.DBImage, job.Metadata)
}
func getDBOFilteredFromJob(job WorkItem, syncID uint64) dbo.FilteredOut {
	ret := dbo.FilteredOut{
		Root:         job.RootName,
		Path:         job.Path,
		Filename:     job.Filename,
		FileSize:     uint64(job.Info.Size()),
		MTime:        job.Info.ModTime().UTC().Truncate(time.Second),
		Ext:          job.Ext,
		FileHash:     job.FileHash,
		MetaHash:     job.FileMetadataHash,
		LastSeenSync: &syncID,
	}
	JSONMetadata, err := json.Marshal(job.Metadata)
	if err == nil {
		ret.ExifJSON = JSONMetadata
	}
	return ret
}

func setJobFromImage(job *WorkItem) {
	if job.DBImage == nil {
		return
	}
	job.Panorama = (job.DBImage.Panorama == 1)
	job.ACLLevel = &job.DBImage.ACLLevel
	job.ACLUser = job.DBImage.ACLUserID
	job.CachedFileHash = job.DBImage.FileHash
	job.CachedFileMetadataHash = job.DBImage.MetaHash
	job.CachedSize = job.DBImage.FileSize
	job.CachedTime = job.DBImage.MTime

	if job.DBImage.ExifJSON != nil {
		metadata := data.Metadata{}
		err := json.Unmarshal(job.DBImage.ExifJSON, &metadata)
		if err != nil {
			log.Logger.Error().Err(err).Str("path", job.RealPath).Msg("error in unmashal")
			return
		}
		job.Metadata = metadata
	}
	job.Source = SourceImages
}

func setJobFromFiltered(job *WorkItem, filtered dbo.FilteredOut) {
	job.CachedFileHash = filtered.FileHash
	job.CachedFileMetadataHash = filtered.MetaHash
	job.CachedSize = filtered.FileSize
	job.CachedTime = filtered.MTime

	if filtered.ExifJSON != nil {
		metadata := data.Metadata{}
		err := json.Unmarshal(filtered.ExifJSON, &metadata)
		if err != nil {
			log.Logger.Error().Err(err).Str("path", job.RealPath).Msg("error in unmashal")
			return
		}
		job.Metadata = metadata
	}
	job.Source = SourceFiltered
}
