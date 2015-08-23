package kitusers

import (
	//"github.com/jinzhu/gorm"
	"github.com/theduke/appkit/users"
)

type User struct {
	users.BaseUserIntID

	AuthData map[string]interface{} `sql:"-" jsonapi:"name=auth-data"` 
}

type Session struct {
	users.BaseSessionIntID
}

type AuthItem struct {
	users.BaseAuthItemIntID
} 
