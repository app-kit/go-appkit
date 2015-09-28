package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Sirupsen/logrus"
	"github.com/theduke/go-apperror"
	"github.com/theduke/go-dukedb/backends/memory"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/app"
	"github.com/theduke/go-appkit/files"
	"github.com/theduke/go-appkit/users"
	"github.com/theduke/go-appkit/users/auth/oauth"
)

func GetNested(rawData interface{}, key string) interface{} {
	data, ok := rawData.(map[string]interface{})
	if !ok {
		return nil
	}

	parts := strings.Split(key, ".")
	if len(parts) > 1 {
		nested := data[parts[0]]
		if nested == nil {
			return nil
		}

		return GetNested(nested, strings.Join(parts[1:], "."))
	} else {
		return data[key]
	}
}

var logMessages []*logrus.Entry

type LoggerHook struct{}

func (h LoggerHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

func (h LoggerHook) Fire(e *logrus.Entry) error {
	logMessages = append(logMessages, e)
	return nil
}

type UserProfile struct {
	users.IntIDUserProfile

	FirstName string
	LastName  string
}

func buildApp() kit.App {
	app := NewApp("")

	conf := app.Config()
	conf.Set("host", "localhost")
	conf.Set("port", 10010)
	conf.Set("url", "http://localhost:10010")

	conf.Set("users.emailConfirmationPath", "?confirm-email={token}")
	conf.Set("users.passwordResetPath", "?reset-password={token}")

	backend := memory.New()
	app.RegisterBackend(backend)

	// Build user service.

	userService := users.NewService(nil, backend, &UserProfile{})
	app.RegisterUserService(userService)

	// Register facebook oauth service.
	appId := "170108689991460"
	appSecret := "cb91c245199a3d3b19fdccbe0c7f93a0"
	fbService := oauth.NewFacebook(appId, appSecret)
	userService.AuthAdaptor("oauth").(*oauth.AuthAdaptorOauth).RegisterService(fbService)

	// Build file service.
	fileHandler := files.NewFileServiceWithFs(nil, "data")
	app.RegisterFileService(fileHandler)

	// Persist log messages in logMessages.
	app.Logger().Hooks.Add(LoggerHook{})

	app.PrepareBackends()

	return app
}

type Data struct {
	Data   interface{}            `json:"data"`
	Errors []apperror.Error       `json:"errors"`
	Meta   map[string]interface{} `json:"meta"`
}

type Client struct {
	Client *http.Client
	Host   string
	Token  string
}

func NewClient(host string) *Client {
	return &Client{
		Client: &http.Client{},
		Host:   host,
	}
}

func (c *Client) SetToken(t string) {
	c.Token = t
}

func (c *Client) DoJson(method, path string, data string) (int, *Data, error) {
	var reader io.Reader
	if data != "" {
		reader = bytes.NewBufferString(data)
	}

	if path[0] != '/' {
		path = "/" + path
	}
	req, err := http.NewRequest(method, c.Host+path, reader)
	if err != nil {
		return 0, nil, err
	}

	if data != "" {
		req.Header.Add("Content-Type", "application/json")
	}
	req.Header.Add("Accept-Encoding", "application/json")

	if c.Token != "" {
		req.Header.Add("Authentication", c.Token)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	if resp.Body == nil {
		return resp.StatusCode, nil, nil
	}
	defer resp.Body.Close()

	rawData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	var respData Data
	if err := json.Unmarshal(rawData, &respData); err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, &respData, nil
}

func (c *Client) GetJson(path string) (int, *Data, error) {
	return c.DoJson("GET", path, "")
}

func (c *Client) PostJson(path string, data string) (int, *Data, error) {
	return c.DoJson("POST", path, data)
}

var _ = Describe("App", func() {
	app := buildApp()
	go func() {
		app.Run()
	}()
	time.Sleep(time.Millisecond * 500)

	client := NewClient("http://localhost:10010")
	var currentUser kit.User
	var currentToken string

	BeforeEach(func() {
		logMessages = nil
	})

	Describe("Usersystem", func() {
		Describe("Password system", func() {
			It("Should create user with password auth", func() {
				js := `{
					"data": {"attributes": {"email": "user1@appkit.com"}},
					"meta": {
						"adaptor": "password", "auth-data": {"password": "test"},
						"profile": {
							"firstName": "First",
							"lastName": "Last"
						}
					}
				}
				`
				status, _, err := client.PostJson("/api/users", js)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(201))

				rawUser, err := app.Backend("memory").FindOneBy("users", "email", "user1@appkit.com")
				Expect(err).ToNot(HaveOccurred())
				Expect(rawUser).ToNot(BeNil())

				plainUser := rawUser.(kit.User)

				// Find full user data with profile and roles.
				user, err := app.UserService().FindUser(plainUser.GetID())
				Expect(err).ToNot(HaveOccurred())
				Expect(user).ToNot(BeNil())

				// Check that user profile was properly created.
				rawProfile := user.GetProfile()
				Expect(rawProfile).ToNot(BeNil())

				profile := rawProfile.(*UserProfile)
				Expect(profile.FirstName).To(Equal("First"))
				Expect(profile.LastName).To(Equal("Last"))

				// Check that confirmation email was sent.
				logEntry := logMessages[len(logMessages)-3]
				Expect(logEntry.Data["action"]).To(Equal("send_email"))
				Expect(logEntry.Data["subject"]).To(Equal("Confirm your Email"))
				Expect(logEntry.Data["to"]).To(Equal([]string{user.GetEmail()}))

				logEntry = logMessages[len(logMessages)-2]
				Expect(logEntry.Data["action"]).To(Equal("users.email_confirmation_mail_sent"))
				Expect(logEntry.Data["user_id"]).To(Equal(user.GetID()))
				Expect(logEntry.Data["email"]).To(Equal(user.GetEmail()))
			})

			It("Should create session for user with password auth", func() {
				js := `{
						"data": {},
						"meta": {"user": "user1@appkit.com", "adaptor": "password", "auth-data": {"password": "test"}}
					}
					`
				status, data, err := client.PostJson("/api/sessions", js)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(201))

				id := GetNested(data.Data, "id").(string)

				token := GetNested(data.Data, "attributes.token").(string)
				Expect(token).ToNot(Equal(""))

				rawSession, err := app.Backend("memory").FindOne("sessions", id)
				Expect(err).ToNot(HaveOccurred())
				Expect(rawSession).ToNot(BeNil())

				session := rawSession.(kit.Session)

				userId := session.GetUserID()
				rawUser, err := app.Backend("memory").FindOne("users", userId)
				Expect(err).ToNot(HaveOccurred())
				Expect(rawUser).ToNot(BeNil())

				user := rawUser.(kit.User)
				Expect(user.GetEmail()).To(Equal("user1@appkit.com"))

				client.SetToken(token)
				currentUser = user
			})

			It("Should resend confirmation email", func() {
				js := `{"data": {}}`
				status, _, err := client.PostJson("/api/method/users.send-confirmation-email", js)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(200))

				// Check that confirmation email was sent.
				logEntry := logMessages[len(logMessages)-2]
				Expect(logEntry.Data["action"]).To(Equal("users.email_confirmation_mail_sent"))
				Expect(logEntry.Data["user_id"]).To(Equal(currentUser.GetID()))
				Expect(logEntry.Data["email"]).To(Equal(currentUser.GetEmail()))

				currentToken = logEntry.Data["token"].(string)
			})

			It("Should confirm email", func() {
				js := fmt.Sprintf(`{"data": {"token": "%v"}}`, currentToken)
				status, _, err := client.PostJson("/api/method/users.confirm-email", js)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(200))

				logEntry := logMessages[len(logMessages)-2]
				Expect(logEntry.Data["action"]).To(Equal("users.email_confirmed"))
				Expect(logEntry.Data["user_id"]).To(Equal(currentUser.GetID()))

				rawUser, err := app.Backend("memory").FindOne("users", currentUser.GetID())
				Expect(rawUser.(kit.User).IsEmailConfirmed()).To(BeTrue())
			})

			It("Should send password reset email", func() {
				js := fmt.Sprintf(`{"data": {"user": "%v"}}`, currentUser.GetEmail())
				status, _, err := client.PostJson("/api/method/users.request-password-reset", js)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(200))

				// Check that confirmation email was sent.
				logEntry := logMessages[len(logMessages)-2]
				Expect(logEntry.Data["action"]).To(Equal("users.password_reset_requested"))
				Expect(logEntry.Data["user_id"]).To(Equal(currentUser.GetID()))
				Expect(logEntry.Data["email"]).To(Equal(currentUser.GetEmail()))

				currentToken = logEntry.Data["token"].(string)
			})

			It("Should reset password", func() {
				js := fmt.Sprintf(`{"data": {"token": "%v", "password": "newpassword"}}`, currentToken)
				status, _, err := client.PostJson("/api/method/users.password-reset", js)
				Expect(err).ToNot(HaveOccurred())
				Expect(status).To(Equal(200))

				logEntry := logMessages[len(logMessages)-2]
				Expect(logEntry.Data["action"]).To(Equal("users.password_reset"))
				Expect(logEntry.Data["user_id"]).To(Equal(currentUser.GetID()))

				// Try to authenticate user with new password.
				user, err := app.Dependencies().UserService().AuthenticateUser(currentUser, "password", map[string]interface{}{"password": "newpassword"})
				Expect(err).ToNot(HaveOccurred())
				Expect(user).ToNot(BeNil())
			})
		})
	})

	Describe("Oauth Facebook", func() {
		userToken := "CAACatoQKRyQBAGzhi73KqSvwTWLip7UG60VUZBdjRhZAEYi71ZAYrtqtKu4d5ib6sVdd8u8K8HeGdg6mBZCqlL4WNkxOxQAmVZA6KzKzCaqZCoxvDGlXLZBvQ5n3KUXgGqtfUSOpmKM4RSd2tx6mTCtXHrHEnpWhyfP7aXErCE0oJFC8uGXLOhzSpComqhvelgZD"

		It("Should create user with facebook connect", func() {
			js := `{
				"data": {"attributes": {"email": "user2@appkit.com"}},
				"meta": {"adaptor": "oauth", "auth-data": {"service": "facebook", "access_token": "%v"}}
			}
			`
			js = fmt.Sprintf(js, userToken)

			status, _, err := client.PostJson("/api/users", js)
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(201))

			rawUser, err := app.Backend("memory").FindOneBy("users", "email", "user2@appkit.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(rawUser).ToNot(BeNil())

			user := rawUser.(kit.User)

			// Check that confirmation email was sent.
			logEntry := logMessages[len(logMessages)-3]
			Expect(logEntry.Data["action"]).To(Equal("send_email"))
			Expect(logEntry.Data["subject"]).To(Equal("Confirm your Email"))
			Expect(logEntry.Data["to"]).To(Equal([]string{user.GetEmail()}))

			logEntry = logMessages[len(logMessages)-2]
			Expect(logEntry.Data["action"]).To(Equal("users.email_confirmation_mail_sent"))
			Expect(logEntry.Data["user_id"]).To(Equal(user.GetID()))
			Expect(logEntry.Data["email"]).To(Equal(user.GetEmail()))
		})

		It("Should create session for user with facebook connect", func() {
			js := `{
					"data": {},
					"meta": {"adaptor": "oauth", "auth-data": {"service": "facebook", "access_token": "%v"}}
				}
			`
			js = fmt.Sprintf(js, userToken)

			status, data, err := client.PostJson("/api/sessions", js)
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(201))

			id := GetNested(data.Data, "id").(string)

			token := GetNested(data.Data, "attributes.token").(string)
			Expect(token).ToNot(Equal(""))

			rawSession, err := app.Backend("memory").FindOne("sessions", id)
			Expect(err).ToNot(HaveOccurred())
			Expect(rawSession).ToNot(BeNil())

			session := rawSession.(kit.Session)

			userId := session.GetUserID()
			rawUser, err := app.Backend("memory").FindOne("users", userId)
			Expect(err).ToNot(HaveOccurred())
			Expect(rawUser).ToNot(BeNil())

			user := rawUser.(kit.User)
			Expect(user.GetEmail()).To(Equal("user2@appkit.com"))

			client.SetToken(token)
			currentUser = user
		})
	})
})
