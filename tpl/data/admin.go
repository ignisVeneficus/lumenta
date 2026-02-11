package data

import (
	"fmt"
	"html/template"
)

type AdminFsContext struct {
	PageContext
	Breadcrumbs Breadcrumbs
	Dirs        AdminFsDirs
	Images      AdminFsImages
}
type AdminFsDirs struct {
	Directories []AdminFsDir
	Paging      Paging
}

func (d AdminFsDirs) Cards() []AdminFsDir {
	return d.Directories
}

type AdminFsDir struct {
	Name   string
	Image  uint64
	ImgQty uint64
	URL    template.URL
}

func (d AdminFsDir) Description() string {
	return ""
}
func (d AdminFsDir) Info() string {
	return fmt.Sprintf("%d images", d.ImgQty)
}

type AdminFsImages struct {
	Images []AdminFsImage
	Paging Paging
}

func (i AdminFsImages) Cards() []AdminFsImage {
	return i.Images
}

type AdminFsImage struct {
	Name     string
	Image    uint64
	URL      template.URL
	ACL      string
	LastSync string
}

func (i AdminFsImage) Description() string {
	return i.ACL
}
func (i AdminFsImage) Info() string {
	return i.LastSync
}
