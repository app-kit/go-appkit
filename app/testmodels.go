package app

import (
	db "github.com/theduke/go-dukedb"
	"github.com/theduke/go-dukedb/models/govalidate"

	"github.com/theduke/go-appkit/files"
	"github.com/theduke/go-appkit/users"
)

type Project struct {
	db.IntIDModel
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

func (t Tag) GetID() string {
	return t.Tag
}

func (t *Tag) SetID(tag string) error {
	t.Tag = tag
	return nil
}

type Todo struct {
	db.IntIDModel
	users.IntUserModel

	Name        string `db:"max:400"`
	Description string `db:"max:400"`

	Files []*files.FileIntID `db:"m2m"`
	Tags  []*Tag             `db:"m2m"`
}

func (t Todo) Collection() string {
	return "todos"
}
