package data

import (
	"net/url"
	"strconv"
)

var (
	ImagePageParam  = "iPage"
	FolderPageParam = "fPage"
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

func (u *URLBuilder) WithImagePaging(page int) *URLBuilder {
	u.query.Set(ImagePageParam, strconv.Itoa(page))
	return u
}
func (u *URLBuilder) WithFolderPaging(page int) *URLBuilder {
	u.query.Set(FolderPageParam, strconv.Itoa(page))
	return u
}
func (u *URLBuilder) WithIntQuery(name string, value int) *URLBuilder {
	u.query.Set(name, strconv.Itoa(value))
	return u
}
