package models

import (
	db "github.com/theduke/go-dukedb"
)

type Address struct {
	Country      string `db:"min:2;max:2;"`
	PostalCode   string `db:"max:100;"`
	State        string `db:"max:100;"`
	Locality     string `db:"max:255;"`
	Street       string `db:"max:500"`
	StreetNumber string `db:"max:100"`
	Top          string `db:"max:100"`
	Description  string `db:"max:1000"`

	Latitude  string `db:"max:100"`
	Longitude string `db:"max:100"`
}

func (a *Address) Collection() string {
	return "addresses"
}

type AddressIntID struct {
	db.IntIDModel
	Address
}

type AddressStrID struct {
	db.StrIDModel
	Address
}
