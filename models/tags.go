package models

import ()

type Tag struct {
	Tag  string `db:"primary-key;max:100"`
	Type string `db:"not-null;max:100"`
}

func (t *Tag) Collection() string {
	return "tags"
}

func (t *Tag) GetID() interface{} {
	return t.Tag
}

func (t *Tag) SetID(tag interface{}) error {
	t.Tag = tag.(string)
	return nil
}

func (t *Tag) GetStrID() string {
	return t.Tag
}

func (t *Tag) SetStrID(tag string) error {
	t.Tag = tag
	return nil
}
