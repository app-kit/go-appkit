package gohtml

import(
	"html/template"
	"fmt"
	"bytes"

	. "github.com/theduke/go-appkit/error"
	"github.com/theduke/go-appkit/templateengines"
)

type Engine struct {
	templates map[string]*template.Template
}

// Ensure Engine implementes TemplateEngine.
var _ templateengines.TemplateEngine = (*Engine)(nil)

func New() *Engine {
	return &Engine{
		templates: make(map[string]*template.Template),
	}
}

func (e *Engine) Build(name, tpl string) (interface{}, Error) {
	t, err := template.New(name).Parse(tpl)
	if err != nil {
		return nil, AppError{
			Code: "tpl_parse_error",
			Message: err.Error(),
		}
	}

	e.templates[name] = t

	return t, nil
}

func (e *Engine) BuildFile(name string, paths ...string) (interface{}, Error) {
	t, err := template.New(name).ParseFiles(paths...)
	if err != nil {
		return nil, AppError{
			Code: "tpl_parse_error",
			Message: err.Error(),
		}
	}

	e.templates[name] = t

	return t, nil
}

func (e *Engine) Get(name string) interface{} {
	return e.templates[name]
}

func (e *Engine) BuildAndRender(name string, tpl string, data interface{})  ([]byte, Error) {
	_, ok := e.templates[name]
	if !ok {
		if _, err := e.Build(name, tpl); err != nil {
			return nil, err
		}
	}

	return e.Render(name, data)
}

func (e *Engine) BuildFileAndRender(name string, data interface{}, paths ...string) ([]byte, Error) {
	_, ok := e.templates[name]
	if !ok {
		if _, err := e.BuildFile(name, paths...); err != nil {
			return nil, err
		}
	}

	return e.Render(name, data)
}

func (e *Engine) Render(name string, data interface{}) ([]byte, Error) {
	t, ok := e.templates[name]
	if !ok {
		return nil, AppError{
			Code: "unknown_template",
			Message: fmt.Sprintf("Template %v was not registered with engine", name),
		}
	}

	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return nil, AppError{
			Code: "tpl_render_error",
			Message: err.Error(),
		}
	}

	return buf.Bytes(), nil
}

// Clean up all templates.
func (e *Engine) Clear() {
	e.templates = make(map[string]*template.Template)
}
