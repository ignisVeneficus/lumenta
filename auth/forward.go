package auth

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/ignisVeneficus/logging"
	authData "github.com/ignisVeneficus/lumenta/auth/data"
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

type ForwardVerifier struct {
	Cidrs        []*net.IPNet
	UserHeader   string
	GroupsHeader string
	AdminRole    string
}

func (fav ForwardVerifier) ContextFromRequest(c context.Context, ip string, request *http.Request) *authData.ACLContext {
	logg, _ := logging.Enter(c, "auth/forward/ctxFromRequest", nil, map[string]any{"ip": ip})
	if !CIDRMatch(fav.Cidrs, ip) {
		logging.Exit(logg, "NOT OK", map[string]any{"problem": "not allowed ip"})
		return nil
	}
	headers := request.Header
	user := headers.Get(fav.UserHeader)
	if user == "" {
		logging.Exit(logg, "NOT OK", map[string]any{"problem": "no user in header"})
		return nil
	}

	role := dbo.RoleUser
	groups := strings.Split(headers.Get(fav.GroupsHeader), ",")
	for _, g := range groups {
		if strings.TrimSpace(g) == fav.AdminRole {
			role = dbo.RoleAdmin
		}
	}
	logging.Exit(logg, "OK", map[string]any{"role": role, "user": &user})
	return &authData.ACLContext{
		ACLContext: dbo.ACLContext{
			Role: role,
		},
		UserName: &user,
		Provider: authData.ProviderForward,
	}

}

func CIDRMatch(cidrs []*net.IPNet, ipString string) bool {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return false
	}
	for _, n := range cidrs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
