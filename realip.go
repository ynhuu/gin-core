package core

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func RealIP(c *gin.Context) string {
	headers := []string{"Cf-Connecting-Ip", "X-Forwarded-For"}

	for _, header := range headers {
		if ip := c.GetHeader(header); ip != "" {
			if strings.Contains(ip, ",") {
				return strings.TrimSpace(strings.Split(ip, ",")[0])
			}
			return strings.TrimSpace(ip)
		}
	}

	return c.ClientIP()
}
