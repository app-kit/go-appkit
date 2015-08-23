package kitusers

import (
	"github.com/jinzhu/gorm"
	"github.com/theduke/gormigrate"
)

func GetMigrations() []*gormigrate.Migration {
	migrations := make([]*gormigrate.Migration, 0)

	v1 := gormigrate.Migration{
		Name: "Creating user tables.",
		Up: func(db *gorm.DB) error {
			db.CreateTable(&User{})
			db.CreateTable(&Session{})
			db.CreateTable(&AuthItem{})

			return nil
		},
		Down: func(db *gorm.DB) error {
			db.DropTableIfExists(&User{})
			db.DropTableIfExists(&Session{})
			db.DropTableIfExists(&AuthItem{})

			return nil
		},
	}
	migrations = append(migrations, &v1)

	return migrations
}
