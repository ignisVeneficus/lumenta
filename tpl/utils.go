package tpl

import (
	"math"

	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/tpl/data"
)

func CreatePaging(url data.URLBuilder, name string, page int, qty uint64, perPage int) data.Paging {
	return data.Paging{
		Url:     url,
		Name:    name,
		ActPage: uint64(page),
		MaxPage: uint64(math.Ceil(float64(qty) / float64(perPage))),
	}
}

func CreateDBOACL(acl authData.ACLContext) dao.ACLContext {
	return dao.ACLContext{
		ViewerUserID: acl.UserID,
		Role:         string(acl.Role),
	}
}
