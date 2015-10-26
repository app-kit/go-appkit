package cms

import (
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/resources"
)

func Build(backend db.Backend, app kit.App, integerIds bool) {

	if integerIds {
		backend.RegisterModel(&TagIntID{})
		backend.RegisterModel(&LocationIntID{})

		backend.RegisterModel(&MenuIntID{})
		backend.RegisterModel(&MenuItemIntID{})
		backend.RegisterModel(&CommentIntID{})
		backend.RegisterModel(&PageComponentIntID{})
		backend.RegisterModel(&PageIntID{})

		app.RegisterResource(resources.NewResource(&TagIntID{}, nil, true))
		app.RegisterResource(resources.NewResource(&LocationIntID{}, nil, true))
		app.RegisterResource(resources.NewResource(&MenuIntID{}, MenuResource{}, true))
		app.RegisterResource(resources.NewResource(&MenuItemIntID{}, MenuItemResource{}, true))
		app.RegisterResource(resources.NewResource(&CommentIntID{}, CommentResource{}, true))
		app.RegisterResource(resources.NewResource(&PageIntID{}, PageResource{}, true))
	} else {
		backend.RegisterModel(&TagStrID{})
		backend.RegisterModel(&LocationStrID{})

		backend.RegisterModel(&MenuStrID{})
		backend.RegisterModel(&MenuItemStrID{})
		backend.RegisterModel(&CommentStrID{})
		backend.RegisterModel(&PageComponentStrID{})
		backend.RegisterModel(&PageStrID{})

		app.RegisterResource(resources.NewResource(&TagStrID{}, nil, true))
		app.RegisterResource(resources.NewResource(&LocationStrID{}, nil, true))
		app.RegisterResource(resources.NewResource(&MenuStrID{}, MenuResource{}, true))
		app.RegisterResource(resources.NewResource(&MenuItemStrID{}, MenuItemResource{}, true))
		app.RegisterResource(resources.NewResource(&CommentStrID{}, CommentResource{}, true))
		app.RegisterResource(resources.NewResource(&PageStrID{}, PageResource{}, true))
	}
}
