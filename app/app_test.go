package app_test

import (
	"net/http"
	"bytes"
	"time"
	"io"
	"io/ioutil"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/theduke/go-dukedb/backends/memory"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/app"
	"github.com/theduke/go-appkit/users"
	"github.com/theduke/go-appkit/files"
)

func buildApp() kit.App {
	app := NewApp("")

	conf := app.Config()
	conf.Set("host", "localhost")
	conf.Set("port", 10010)

	backend := memory.New()
	app.RegisterBackend("memory", backend)

	userService := users.NewService(nil)
	app.RegisterUserService(userService)

	fileHandler := files.NewFileServiceWithFs("data")
	app.RegisterFileService(fileHandler)

	app.PrepareBackends()

	return app
}

type Data struct {
	Data interface{} `json:"data"`
	Errors []error `json:"errors"`
	Meta map[string]interface{} `json:"meta"`
}

type Client struct {
	Client *http.Client
	Host string
}

func NewClient(host string) *Client {
	return &Client{
		Client: &http.Client{},
		Host: host,
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
	req, err := http.NewRequest(method, c.Host + path, reader)
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

			user, err := app.Backend("memory").FindOneBy("users", "email", "user1@appkit.com")
			Expect(err).ToNot(HaveOccurred())
			Expect(user).ToNot(BeNil())
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
