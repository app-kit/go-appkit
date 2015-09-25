package cms

import (
	"github.com/theduke/go-appkit/resources"
)

/*
	db "github.com/theduke/go-dukedb"
	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/files"
*/

type MenuResource struct {
	resources.AdminResource
}

type MenuItemResource struct {
	resources.AdminResource
}

type CommentResource struct {
}

type PageResource struct {
	resources.UserResource
}
