package gorm

import (
	"github.com/gin-gonic/gin"
	ogorm "github.com/jinzhu/gorm"
)

func DbProviderMiddleware(db *ogorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}
