package scene

import "github.com/gin-gonic/gin"

type LoginScene struct {
	Scene
	LoginURL          string
	ForgotPasswordURL string
	Next              string
}

func (s Scene) Login(c *gin.Context) *LoginScene {
	// Return the login scene with the default URLs set.
	return &LoginScene{
		Scene:             s,
		LoginURL:          "/v1/login",
		ForgotPasswordURL: "/forgot-password",
		Next:              c.Query("next"),
	}
}
