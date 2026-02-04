package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/ignisVeneficus/lumenta/auth/data"
)

const AuthContextKey = "auth"

func GetAuthContex(c *gin.Context) data.ACLContext {
	acl, ok := c.Get(AuthContextKey)
	if ok {
		return acl.(data.ACLContext)
	}
	return *data.GuestContext()
}
func SetAuthContex(c *gin.Context, ctx data.ACLContext) {
	c.Set(AuthContextKey, ctx)
}
