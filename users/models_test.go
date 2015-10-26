package users_test

import (
	. "github.com/app-kit/go-appkit/users"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Models", func() {

	Describe("User", func() {

		var user *User

		BeforeEach(func() {
			user = &User{}
		})

		It("Should add roles", func() {
			user.AddRole("role1", "role2")
			Expect(user).To(Equal(&User{
				Roles: []*Role{&Role{Name: "role1"}, &Role{Name: "role2"}},
			}))
		})

		It("Should set roles", func() {
			user.SetRoles([]string{"role1", "role2"})
			Expect(user).To(Equal(&User{
				Roles: []*Role{&Role{Name: "role1"}, &Role{Name: "role2"}},
			}))
		})

		It("Should get roles", func() {
			user.AddRole("role1", "role2")
			Expect(user.GetRoles()).To(Equal([]string{"role1", "role2"}))
		})

		It("Should clear roles", func() {
			user.AddRole("role1", "role2")
			user.ClearRoles()
			Expect(user.GetRoles()).To(Equal([]string{}))
		})

		It("Should remove role", func() {
			user.AddRole("role1", "role2", "role3")
			user.RemoveRole("role1", "role3")
			Expect(user.GetRoles()).To(Equal([]string{"role2"}))
		})

		It("Should .HasRole()", func() {
			user.AddRole("role1", "role2")
			Expect(user.HasRole("role1")).To(BeTrue())
			Expect(user.HasRole("role3")).To(BeFalse())
			Expect(user.HasRole("role1", "role3")).To(BeTrue())
			Expect(user.HasRole("role4", "role3")).To(BeFalse())
		})

		It("Should .HasPermission", func() {
			u := &User{
				Roles: []*Role{
					&Role{
						Name: "r1",
						Permissions: []*Permission{
							&Permission{Name: "perm1"},
							&Permission{Name: "perm2"},
						},
					},
				},
			}

			Expect(u.HasPermission("perm1")).To(BeTrue())
			Expect(u.HasPermission("perm1", "perm3")).To(BeTrue())
			Expect(u.HasPermission("perm3", "perm4")).To(BeFalse())
		})
	})

	Describe("Role", func() {
		var role *Role

		BeforeEach(func() {
			role = &Role{}
		})

		It("Should .SetPermissions()", func() {
			role.SetPermissions([]string{"perm1", "perm2"})
			Expect(role).To(Equal(&Role{
				Permissions: []*Permission{
					&Permission{Name: "perm1"},
					&Permission{Name: "perm2"},
				},
			}))
		})

		It("Should .AddPermission()", func() {
			role.AddPermission("perm1", "perm2")
			Expect(role).To(Equal(&Role{
				Permissions: []*Permission{
					&Permission{Name: "perm1"},
					&Permission{Name: "perm2"},
				},
			}))
		})

		It("Should .GetPermissions", func() {
			role.AddPermission("perm1", "perm2")
			Expect(role.GetPermissions()).To(Equal([]string{"perm1", "perm2"}))
		})

		It("Should .ClearPermissions()", func() {
			role.AddPermission("perm1", "perm2")
			role.ClearPermissions()
			Expect(role.GetPermissions()).To(Equal([]string{}))
		})

		It("SHould .RemovePermission()", func() {
			role.AddPermission("perm1", "perm2", "perm3")
			role.RemovePermission("perm1", "perm3")
			Expect(role.GetPermissions()).To(Equal([]string{"perm2"}))
		})

		It("Should .HasPermissions", func() {
			role.AddPermission("perm1", "perm2", "perm3")
			Expect(role.HasPermission("perm1")).To(BeTrue())
			Expect(role.HasPermission("perm1", "perm2")).To(BeTrue())
			Expect(role.HasPermission("perm5", "perm2")).To(BeTrue())
			Expect(role.HasPermission("perm6")).To(BeFalse())
			Expect(role.HasPermission("perm6", "perm6")).To(BeFalse())
		})
	})

})
