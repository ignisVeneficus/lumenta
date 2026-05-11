package auth

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth/data"
)

const AuthContextKey = "auth"

func GetAuthContex(c context.Context) data.ACLContext {
	acl := c.Value(AuthContextKey)
	if acl != nil {
		return acl.(data.ACLContext)
	}
	return *data.GuestContext()
}
func SetAuthContex(c *gin.Context, ctx data.ACLContext) {
	c.Set(AuthContextKey, ctx)
}
