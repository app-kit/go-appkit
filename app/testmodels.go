package app

import (
	db "github.com/theduke/go-dukedb"
	"github.com/theduke/go-dukedb/models/govalidate"

	"github.com/app-kit/go-appkit/files"
	"github.com/app-kit/go-appkit/users"
)

type Project struct {
	db.IntIdModel
	users.IntUserModel
	govalidate.Model

	Name        string `db:"max:400" valid:"required"`
	Description string `db:"max:4000" valid:"-"`

	Todos []*Todo `valid:"-"`
}

func (p Project) Collection() string {
	return "projects"
}

type Tag struct {
	Tag string `db:"primary-key;max:100"`
}

func (t Tag) Collection() string {
	return "tags"
}

func (t Tag) GetId() string {
	return t.Tag
}

func (t *Tag) SetId(tag string) error {
	t.Tag = tag
	return nil
}

type Todo struct {
	db.IntIdModel
	users.IntUserModel

	Name        string `db:"max:400"`
	Description string `db:"max:400"`

	Files []*files.FileIntId `db:"m2m"`
	Tags  []*Tag             `db:"m2m"`
}

func (t Todo) Collection() string {
	return "todos"
}
