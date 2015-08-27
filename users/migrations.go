package users

import (
	db "github.com/theduke/dukedb"
	kit "github.com/theduke/appkit"
)

func GetUserMigrations(app *kit.App) []db.Migration {
	migrations := make([]db.Migration, 0)

	v1 := db.Migration{
		Name: "Create User, UserProfile, Session, AuthItem, Roles and Permissions tables",
		Up: func(b db.MigrationBackend) error {
			if err := b.CreateCollection("users"); err != nil {
				return err
			}
			if err := b.CreateCollection("sessions"); err != nil {
				return err
			}
			if err := b.CreateCollection("auth_items"); err != nil {
				return err
			}
			if err := b.CreateCollection("roles"); err != nil {
				return err
			}
			if err := b.CreateCollection("permissions"); err != nil {
				return err
			}

			return nil
		},
	}
	migrations = append(migrations, v1)

	v2 := db.Migration{
		Name: "Create admin user",
		Up: func(b db.MigrationBackend) error {
			userHandler := app.GetUserHandler()
			user := userHandler.GetUserResource().NewModel().(kit.ApiUser)
			user.SetUsername("admin")
			user.SetEmail("admin@admin.com")

			userHandler.CreateUser(user, "password", map[string]interface{}{"password": "admin"})

			return nil
		},
	}
	migrations = append(migrations, v2)

	return migrations
}
