package data

import (
	"github.com/ignisVeneficus/lumenta/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

type PageImage struct {
	Image     dbo.Image
	SingleMap *SingleMap
	Metadata  Metadata
	Tags      data.Forest[*ViewTreeNode]
}

type Metadata struct {
	Title          string
	Description    string
	Blocks         []MetadataBlock
	MetadataValues []MetadataValue
}

type MetadataValue struct {
	Label string
	Value string
}
type MetadataBlock struct {
	Label string
	Data  []string
}

func (mb MetadataBlock) IsEmpty() bool {
	if len(mb.Data) == 0 {
		return true
	}
	for _, d := range mb.Data {
		if d != "" {
			return false
		}
	}
	return true
}
