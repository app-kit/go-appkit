package oauth_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/theduke/go-appkit/users/auth/oauth"
)

var _ = Describe("Facebook", func() {
	appId := "170108689991460"
	appSecret := "cb91c245199a3d3b19fdccbe0c7f93a0"
	userToken := "CAACatoQKRyQBAGzhi73KqSvwTWLip7UG60VUZBdjRhZAEYi71ZAYrtqtKu4d5ib6sVdd8u8K8HeGdg6mBZCqlL4WNkxOxQAmVZA6KzKzCaqZCoxvDGlXLZBvQ5n3KUXgGqtfUSOpmKM4RSd2tx6mTCtXHrHEnpWhyfP7aXErCE0oJFC8uGXLOhzSpComqhvelgZD"
	var appToken string

	Describe("Source", func() {
		source := NewFacebook(appId, appSecret)

		It("Should .Exchange()", func() {
			token, err := source.Exchange(userToken)
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(Equal(""))

			appToken = token
		})

		It("Should .GetUserData()", func() {
			if appToken == "" {
				Skip("no token")
			}
			data, err := source.GetUserData(appToken)
			Expect(err).ToNot(HaveOccurred())
			Expect(data).ToNot(BeNil())
		})
	})
})
