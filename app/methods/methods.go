package methods

import (
	kit "github.com/theduke/go-appkit"
)

type Method struct {
	name         string
	isBlocking   bool
	requiresUser bool
	run          func(a kit.App, r kit.Request, unblock func()) kit.Response
}

func NewMethod(name string, blocking bool, run func(a kit.App, r kit.Request, unblock func()) kit.Response) *Method {
	return &Method{
		name:       name,
		isBlocking: blocking,
		run:        run,
	}
}

func (m Method) Name() string {
	return m.name
}

func (m Method) IsBlocking() bool {
	return m.isBlocking
}

func (m Method) RequiresUser() bool {
	return m.requiresUser
}

func (m Method) Run(a kit.App, r kit.Request, unblock func()) kit.Response {
	return m.run(a, r, unblock)
}
