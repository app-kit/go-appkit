package cms

import (
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/resources"
)

func Build(backend db.Backend, app kit.App, integerIds bool) {

	if integerIds {
		backend.RegisterModel(&TagIntId{})
		backend.RegisterModel(&LocationIntId{})

		backend.RegisterModel(&MenuIntId{})
		backend.RegisterModel(&MenuItemIntId{})
		backend.RegisterModel(&CommentIntId{})
		backend.RegisterModel(&PageComponentIntId{})
		backend.RegisterModel(&PageIntId{})

		app.RegisterResource(resources.NewResource(&TagIntId{}, nil, true))
		app.RegisterResource(resources.NewResource(&LocationIntId{}, nil, true))
		app.RegisterResource(resources.NewResource(&MenuIntId{}, MenuResource{}, true))
		app.RegisterResource(resources.NewResource(&MenuItemIntId{}, MenuItemResource{}, true))
		app.RegisterResource(resources.NewResource(&CommentIntId{}, CommentResource{}, true))
		app.RegisterResource(resources.NewResource(&PageIntId{}, PageResource{}, true))
	} else {
		backend.RegisterModel(&TagStrId{})
		backend.RegisterModel(&LocationStrId{})

		backend.RegisterModel(&MenuStrId{})
		backend.RegisterModel(&MenuItemStrId{})
		backend.RegisterModel(&CommentStrId{})
		backend.RegisterModel(&PageComponentStrId{})
		backend.RegisterModel(&PageStrId{})

		app.RegisterResource(resources.NewResource(&TagStrId{}, nil, true))
		app.RegisterResource(resources.NewResource(&LocationStrId{}, nil, true))
		app.RegisterResource(resources.NewResource(&MenuStrId{}, MenuResource{}, true))
		app.RegisterResource(resources.NewResource(&MenuItemStrId{}, MenuItemResource{}, true))
		app.RegisterResource(resources.NewResource(&CommentStrId{}, CommentResource{}, true))
		app.RegisterResource(resources.NewResource(&PageStrId{}, PageResource{}, true))
	}
}
