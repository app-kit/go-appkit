package gin

import (
	"errors"
	"net/url"

	"github.com/gin-gonic/gin"
)

/**
 * Send an error response.
 */
func ErrorResponse(code int, message string, data *gin.H, c *gin.Context) {
	c.JSON(code, gin.H{
		"error":   message,
		"success": false,
		"data":    data,
	})

	c.Abort()
}

/**
 * Check if a string slice contains a value.
 */
func SliceContainsString(slice []string, needle string) bool {
	for _, val := range slice {
		if val == needle {
			return true
		}
	}
	return false
}

// Check that all keys in vals are either in allowed or ignored slices.
func CheckAllKeysAllowed(vals url.Values, allowed []string, ignored []string) error {
	for key := range vals {
		if !(SliceContainsString(allowed, key) || SliceContainsString(ignored, key)) {
			return errors.New("invalid_key")
		}
	}

	return nil
}
