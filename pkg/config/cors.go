package config

import (
	"time"

	"github.com/gin-contrib/cors"
)

func (c Config) CORS() cors.Config {
	// Create a CORS config with the configured allowed origins.
	return cors.Config{
		AllowOrigins:     c.AllowOrigins,
		AllowMethods:     []string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-CSRF-TOKEN"},
		ExposeHeaders:    []string{"Content-Length", "Access-Control-Allow-Origin"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}
