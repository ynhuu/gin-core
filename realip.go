package core

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func RealIP(c *gin.Context) string {
	headers := []string{"Cf-Connecting-Ip", "X-Forwarded-For"}

	for _, header := range headers {
		if ip := c.GetHeader(header); ip != "" {
			first, _, _ := strings.Cut(ip, ",")
			return strings.TrimSpace(first)
		}
	}

	return c.ClientIP()
}
