package methods

import (
	kit "github.com/app-kit/go-appkit"
)

type Method struct {
	Name     string
	Blocking bool
	Handler  kit.MethodHandler
}

// Ensure Method implements kit.Method interface.
var _ kit.Method = (*Method)(nil)

func (m Method) GetName() string {
	return m.Name
}

func (m Method) IsBlocking() bool {
	return m.Blocking
}

func (m Method) GetHandler() kit.MethodHandler {
	return m.Handler
}
