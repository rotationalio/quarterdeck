package api

import gimauth "go.rtnl.ai/gimlet/auth"

// Model for an OIDC 'userinfo' endpoint response.
// See: https://openid.net/specs/openid-connect-core-1_0.html#UserInfo
type UserInfo struct {
	Subject       string `json:"sub,omitempty"`
	Name          string `json:"name,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
}

// Converts a JWT claims into a UserInfo.
func NewUserInfo(claims *gimauth.Claims) *UserInfo {
	return &UserInfo{
		Subject:       claims.Subject,
		Name:          claims.Name,
		Email:         claims.Email,
		EmailVerified: true,
	}
}
