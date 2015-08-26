package appkit

import (

)

type Method struct {
	Name string
	RequiresUser bool

	Run func(a *App, r *Request) (interface{}, ApiError)
}
