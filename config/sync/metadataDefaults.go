package sync

import "github.com/ignisVeneficus/lumenta/data"

func DefaultDBMetadataConfig() MetadataConfig {
	return MetadataConfig{
		Fields: map[string]MetadataFieldConfig{

			// =========================
			// EXIF – idő és technika
			// =========================

			data.MetaTakenAt: {
				Sources: []MetadataSourceConfig{
					{Ref: "composite:DateTimeCreated"},
					{Ref: "xmp-exif:DateTimeOriginal"},
					{Ref: "exififd:DateTimeOriginal"},
					{Ref: "exififd:createdate"},
					{Ref: "exif:DateTimeOriginal"},
					{Ref: "exif:CreateDate"},
				},
				Type: data.MetaDateTime,
			},

			data.MetaCamera: {
				Sources: []MetadataSourceConfig{
					{Ref: "ifd0:model"},
					{Ref: "exif:Model"},
				},
				Type: data.MetaString,
			},
			data.MetaMaker: {
				Sources: []MetadataSourceConfig{
					{Ref: "ifd0:make"},
				},
				Type: data.MetaString,
			},

			data.MetaLens: {
				Sources: []MetadataSourceConfig{
					{Ref: "composite:LensID"},
					{Ref: "exififd:LensModel"},
					{Ref: "exififd:LensInfo"},
					{Ref: "xmp-aux:Lens"},
				},
				Type: data.MetaString,
			},

			data.MetaFocalLength: {
				Sources: []MetadataSourceConfig{
					{Ref: "exififd:FocalLength"},
				},
				Type: data.MetaFloat, // rational → float
				Unit: "mm",
			},

			data.MetaAperture: {
				Sources: []MetadataSourceConfig{
					{Ref: "composite:aperture"},
					{Ref: "exififd:fnumber"},
					{Ref: "exififd:aperturevalue"},
				},
				Type: data.MetaFloat,
				Unit: "f",
			},

			data.MetaISO: {
				Sources: []MetadataSourceConfig{
					{Ref: "exififd:ISO"},
					{Ref: "exif:ISO"},
				},
				Type: data.MetaInt,
				Unit: "ISO",
			},
			data.MetaExposureTime: {
				Sources: []MetadataSourceConfig{
					{Ref: "Composite:ShutterSpeed"},
					{Ref: "exififd:ExposureTime"},
				},
				Type: data.MetaString,
				Unit: "s",
			},

			// =========================
			// GPS
			// =========================

			data.MetaLatitude: {
				Sources: []MetadataSourceConfig{
					{Ref: "Composite:GPSLatitude"},
					{Ref: "exif:GPSLatitude"},
				},
				Type: data.MetaString,
			},

			data.MetaLongitude: {
				Sources: []MetadataSourceConfig{
					{Ref: "composite:GPSLongitude"},
					{Ref: "exif:GPSLongitude"},
				},
				Type: data.MetaString,
			},

			// =========================
			// Orientáció / rating
			// =========================

			data.MetaRotation: {
				Sources: []MetadataSourceConfig{
					{Ref: "ifd0:orientation"},
					{Ref: "exif:Orientation"},
				},
				Type: data.MetaString,
			},

			data.MetaRating: {
				Sources: []MetadataSourceConfig{
					{Ref: "xmp-xmp:Rating"},
				},
				Type: data.MetaInt,
			},

			// =========================
			// Emberi leírás
			// =========================

			data.MetaTitle: {
				Sources: []MetadataSourceConfig{
					{Ref: "xmp-dc:title"},
					{Ref: "iptc:objectName"},
				},
				Type: data.MetaString,
			},

			data.MetaSubject: {
				Sources: []MetadataSourceConfig{
					{Ref: "xmp-dc:description"},
					{Ref: "iptc:caption"},
				},
				Type: data.MetaString,
			},

			// =========================
			// TAGS – különleges eset
			// =========================

			data.MetaTags: {
				Sources: []MetadataSourceConfig{
					{Ref: "xmp-digikam:tagslist"},
					{Ref: "xmp-lr:hierarchicalsubject"},
					{Ref: "iptc:Keywords"},
					{Ref: "xmp-dc:subject"},
				},
				Type: data.MetaList,
			},
			// =========================
			// IMAGE SIZE
			// =========================

			data.MetaHeight: {
				Sources: []MetadataSourceConfig{
					{Ref: "file:ImageHeight"},
					{Ref: "XMP-tiff:ImageHeight"},
					{Ref: "exif:PixelYDimension"},
				},
				Type: data.MetaInt,
			},
			data.MetaWidth: {
				Sources: []MetadataSourceConfig{
					{Ref: "file:ImageWidth"},
					{Ref: "XMP-tiff:ImageWidth"},
					{Ref: "exif:PixelXDimension"},
				},
				Type: data.MetaInt,
			},
			/*
				data.MetaSize: {
					Sources: []MetadataSourceConfig{
						{Ref: "Composite:ImageSize"},
					},
					Type: data.MetaString,
				},
			*/
		},
	}
}
