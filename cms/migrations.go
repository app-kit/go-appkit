package cms

import (
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
)

func BuildMigrations(b db.MigrationBackend, app kit.App) []db.Migration {
	migrations := make([]db.Migration, 0)

	v1 := db.Migration{
		Name:            "Create all CMS tables.",
		WrapTransaction: true,
		Up: func(b db.MigrationBackend) error {
			if err := b.CreateCollection(
				"files",
				"tags",
				"menus",
				"menu_items",
				"comments",
				"locations",
				"pages"); err != nil {
				return err
			}

			return nil
		},
	}
	migrations = append(migrations, v1)

	return migrations
}
