package middleware

import (
	"bbgre/utils"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.Request.Header.Get("x-token")
		if tokenString == "" {
			c.JSON(401, gin.H{"error": "Authorization header is missing"})
			c.Abort()
			return
		}

		claims, err := utils.ParseToken(tokenString)

		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
			return
		} else {
			c.Set("userID", claims.UserID)
		}

		c.Next()
	}
}
