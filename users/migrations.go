package users

import (
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

func GetUserMigrations(app kit.App) []db.Migration {
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

			if userService := app.UserService(); userService != nil {
				if profile := userService.ProfileModel(); profile != nil {
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
			userService := app.UserService()

			permissions := userService.PermissionResource()
			allPerm := &Permission{Name: "all"}
			if err := permissions.Create(allPerm, nil); err != nil {
				return err
			}

			// Create admin role.
			adminRole := &Role{Name: "admin"}
			adminRole.Permissions = []*Permission{allPerm}
			roles := userService.RoleResource()
			if err := roles.Create(adminRole, nil); err != nil {
				return err
			}

			user := userService.UserResource().NewModel().(kit.User)
			user.SetUsername("admin")
			user.SetEmail("admin@admin.com")
			user.AddRole(adminRole)

			userService.CreateUser(user, "password", map[string]interface{}{"password": "admin"})

			return nil
		},
	}
	migrations = append(migrations, v2)

	return migrations
}
