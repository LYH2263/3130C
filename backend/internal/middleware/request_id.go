package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = generateRequestID()
		}
		c.Set(RequestIDKey, rid)
		c.Writer.Header().Set("X-Request-ID", rid)
		c.Next()
	}
}

func GetRequestID(c *gin.Context) string {
	v, ok := c.Get(RequestIDKey)
	if !ok {
		return ""
	}
	rid, ok := v.(string)
	if !ok {
		return ""
	}
	return rid
}

func generateRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}
