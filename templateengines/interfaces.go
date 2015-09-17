package templateengines

import(
	. "github.com/theduke/go-appkit/error"
)

type TemplateEngine interface {
	Build(name string, tpl string) (interface{}, Error)
	BuildFile(name string, paths ...string) (interface{}, Error)

	Get(name string) interface{}

	BuildAndRender(name string, tpl string, data interface{})  ([]byte, Error)
	BuildFileAndRender(name string, data interface{}, paths ...string) ([]byte, Error)

	Render(name string, data interface{}) ([]byte, Error)

	// Clean up all templates.
	Clear()
}
