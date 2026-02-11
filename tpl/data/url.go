package data

import (
	"net/url"
	"strconv"
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

func (u *URLBuilder) WithPaging(page int) *URLBuilder {
	u.query.Set(QueryPaging, strconv.Itoa(page))
	return u
}
func (u *URLBuilder) WithIntQuery(name string, value int) *URLBuilder {
	u.query.Set(name, strconv.Itoa(value))
	return u
}
