package data

import (
	"net/url"
	"strconv"
)

var (
	ImagePageParam  = "iPage"
	FolderPageParam = "fPage"
	SyncPageParam   = "sPage"
	SearchParam     = "q"
	FilterParam     = "f"
)

type URLBuilder struct {
	path  string
	query url.Values
}

func NewURL(path string) *URLBuilder {
	return &URLBuilder{
		path:  path,
		query: make(url.Values),
	}
}
func (u *URLBuilder) String() string {
	if len(u.query) == 0 {
		return u.path
	}
	return u.path + "?" + u.query.Encode()
}
func (u *URLBuilder) WithParam(name string, value string) *URLBuilder {
	u.query.Set(name, value)
	return u
}
func (u *URLBuilder) WithArrayParams(name string, value []string) *URLBuilder {
	for _, v := range value {
		u.query.Set(name, v)
	}
	return u
}

func (u *URLBuilder) WithImagePaging(page uint64) *URLBuilder {
	u.query.Set(ImagePageParam, strconv.FormatUint(page, 10))
	return u
}
func (u *URLBuilder) WithImageIntPaging(page int) *URLBuilder {
	u.query.Set(ImagePageParam, strconv.Itoa(page))
	return u
}
func (u *URLBuilder) WithFolderPaging(page uint64) *URLBuilder {
	u.query.Set(FolderPageParam, strconv.FormatUint(page, 10))
	return u
}
func (u *URLBuilder) WithIntQuery(name string, value int) *URLBuilder {
	u.query.Set(name, strconv.Itoa(value))
	return u
}
func (u *URLBuilder) WithUintQuery(name string, value uint64) *URLBuilder {
	u.query.Set(name, strconv.FormatUint(value, 10))
	return u
}
