package users

import (
	db "github.com/theduke/go-dukedb"

	kit "github.com/app-kit/go-appkit"
)

func GetUserMigrations(service kit.UserService) []db.Migration {
	migrations := make([]db.Migration, 0)

	v1 := db.Migration{
		Name:            "Create user system tables",
		WrapTransaction: true,
		Up: func(b db.MigrationBackend) error {
			if err := b.CreateCollection(
				"user_permissions",
				"user_roles",
				"users",
				"sessions",
				"user_tokens",
				"users_auth_passwords",
				"users_auth_oauth"); err != nil {
				return err
			}

			if service != nil {
				if profiles := service.ProfileResource(); profiles != nil {
					if err := b.CreateCollection(profiles.Collection()); err != nil {
						return err
					}
				}
			}

			return nil
		},
	}
	migrations = append(migrations, v1)

	v2 := db.Migration{
		Name: "Create admin role and user",
		Up: func(b db.MigrationBackend) error {
			allPerm := &Permission{Name: "all"}
			if err := b.Create(allPerm); err != nil {
				return err
			}

			// Create admin role.
			adminRole := &Role{Name: "admin"}
			adminRole.Permissions = []*Permission{allPerm}
			if err := b.Create(adminRole); err != nil {
				return err
			}

			user := service.UserResource().CreateModel().(kit.User)
			user.SetUsername("admin")
			user.SetEmail("admin@admin.com")
			user.AddRole("admin")

			err := service.CreateUser(user, "password", map[string]interface{}{"password": "admin"})
			if err != nil {
				return err
			}

			return nil
		},
	}
	migrations = append(migrations, v2)

	return migrations
}
