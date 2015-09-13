package users

import (
	kit "github.com/theduke/go-appkit"
	db "github.com/theduke/go-dukedb"
)

func GetUserMigrations(app *kit.App) []db.Migration {
	migrations := make([]db.Migration, 0)

	v1 := db.Migration{
		Name: "Create user system tables",
		Up: func(b db.MigrationBackend) error {
			if err := b.CreateCollection("permissions"); err != nil {
				return err
			}
			if err := b.CreateCollection("roles"); err != nil {
				return err
			}
			if err := b.CreateCollection("users"); err != nil {
				return err
			}
			if err := b.CreateCollection("sessions"); err != nil {
				return err
			}
			if err := b.CreateCollection("auth_items"); err != nil {
				return err
			}

			if userHandler := app.GetUserHandler(); userHandler != nil {
				if profile := userHandler.GetProfileModel(); profile != nil {
					b.CreateCollection(profile.Collection())
				}
			}

			return nil
		},
	}
	migrations = append(migrations, v1)

	v2 := db.Migration{
		Name: "Create admin role and user",
		Up: func(b db.MigrationBackend) error {
			userHandler := app.GetUserHandler()

			permissions := userHandler.GetPermissionResource()
			allPerm := &Permission{Name: "all"}
			if err := permissions.Create(allPerm, nil); err != nil {
				return err
			}

			// Create admin role.
			adminRole := &Role{Name: "admin"}
			adminRole.Permissions = []*Permission{allPerm}
			roles := userHandler.GetRoleResource()
			if err := roles.Create(adminRole, nil); err != nil {
				return err
			}

			user := userHandler.GetUserResource().NewModel().(kit.ApiUser)
			user.SetUsername("admin")
			user.SetEmail("admin@admin.com")
			user.AddRole(adminRole)

			userHandler.CreateUser(user, "password", map[string]interface{}{"password": "admin"})

			return nil
		},
	}
	migrations = append(migrations, v2)

	return migrations
}
