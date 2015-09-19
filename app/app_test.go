package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Sirupsen/logrus"
	"github.com/theduke/go-dukedb/backends/memory"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/app"
	"github.com/theduke/go-appkit/files"
	"github.com/theduke/go-appkit/users"
)

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

func buildApp() kit.App {
	app := NewApp("")

	conf := app.Config()
	conf.Set("host", "localhost")
	conf.Set("port", 10010)
	conf.Set("url", "http://localhost:10010")

	conf.Set("users.emailConfirmationPath", "?confirm-email={token}")

	backend := memory.New()
	app.RegisterBackend(backend)

	userService := users.NewService(nil, nil)
	app.RegisterUserService(userService)

	fileHandler := files.NewFileServiceWithFs(nil, "data")
	app.RegisterFileService(fileHandler)

	// Persist log messages in logMessages.
	app.Logger().Hooks.Add(LoggerHook{})

	app.PrepareBackends()

	return app
}

type Data struct {
	Data   interface{}            `json:"data"`
	Errors []kit.AppError         `json:"errors"`
	Meta   map[string]interface{} `json:"meta"`
}

type Client struct {
	Client *http.Client
	Host   string
}

func NewClient(host string) *Client {
	return &Client{
		Client: &http.Client{},
		Host:   host,
	}
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

	resp, err := c.Client.Do(req)
	if err != nil {
		fmt.Printf("request error: %v", err)
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

	BeforeEach(func() {
		logMessages = nil
	})

	Describe("Usersystem", func() {
		It("Should create user with password auth", func() {
			js := `{
				"data": {"attributes": {"email": "user1@appkit.com"}},
				"meta": {"adaptor": "password", "auth-data": {"password": "test"}}
			}
			`
			status, _, err := client.PostJson("/api/users", js)
			Expect(err).ToNot(HaveOccurred())
			Expect(status).To(Equal(201))

			rawUser, err := app.Backend("memory").FindOneBy("users", "email", "user1@appkit.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(rawUser).ToNot(BeNil())

			user := rawUser.(kit.User)

			// Check that confirmation email was sent.
			logEntry := logMessages[len(logMessages)-1]
			Expect(logEntry.Data["action"]).To(Equal("send_email"))
			Expect(logEntry.Data["subject"]).To(Equal("Confirm your Email"))
			Expect(logEntry.Data["to"]).To(Equal([]string{user.GetEmail()}))
		})
	})

	It("Should create session for user", func() {
		js := `{
				"data": {},
				"meta": {"user": "user1@appkit.com", "adaptor": "password", "auth-data": {"password": "test"}}
			}
			`
		status, rawData, err := client.PostJson("/api/sessions", js)
		Expect(err).ToNot(HaveOccurred())
		Expect(status).To(Equal(201))

		data := rawData.Data.(map[string]interface{})
		id := data["id"].(string)

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
	})
})
