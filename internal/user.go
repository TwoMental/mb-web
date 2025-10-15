package internal

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	userKey = "UserID"
)

func generateUserId() string {
	// Generate a random user ID
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func GetUserID(c *gin.Context) string {
	// Try to read the user ID from the request cookie
	userId, err := c.Cookie(userKey)
	if err != nil || userId == "" {
		// Cookie is missing, generate a new user ID
		userId = generateUserId() // Generate the user ID
		// Set cookie
		c.SetCookie(userKey, userId, 10800, "/", "", false, true)
	}
	return userId
}
