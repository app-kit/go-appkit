package oauth

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/theduke/go-apperror"
)

type Service interface {
	Name() string

	GetClientID() string
	GetClientSecrect() string
	GetAuthUrl() string
	GetRedirectUrl() string
	GetTokenUrl() string
	GetExchangeUrl(token string) string
	GetEndpointUrl() string

	Exchange(token string) (string, apperror.Error)
	GetClient(token string) Client
	GetUserData(token string) (*UserData, apperror.Error)
}

type BaseService struct {
	AuthUrl     string
	RedirectUrl string
	TokenUrl    string
	EndpointUrl string

	ClientID     string
	ClientSecret string
}

func (s *BaseService) GetClientID() string {
	return s.ClientID
}

func (s *BaseService) GetClientSecrect() string {
	return s.ClientSecret
}

func (s *BaseService) GetAuthUrl() string {
	return s.AuthUrl
}

func (s *BaseService) GetRedirectUrl() string {
	return s.RedirectUrl
}

func (s *BaseService) GetTokenUrl() string {
	return s.TokenUrl
}

func (s *BaseService) GetEndpointUrl() string {
	return s.EndpointUrl
}

func exchangeToken(url string) (string, apperror.Error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", apperror.Wrap(err, "http_error", "")
	}

	if resp.Body == nil {
		return "", apperror.New("http_no_body")
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", apperror.Wrap(err, "body_read_error", "")
	}

	return strings.TrimSpace(string(content)), nil
}

type Facebook struct {
	BaseService
}

// Ensure Facebook implements Source.
var _ Service = (*Facebook)(nil)

func NewFacebook(clientId, clientSecret string) *Facebook {
	return &Facebook{
		BaseService{
			AuthUrl:     "https://www.facebook.com/dialog/oauth",
			RedirectUrl: "",
			TokenUrl:    "https://graph.facebook.com/oauth/access_token",
			EndpointUrl: "https://graph.facebook.com",

			ClientID:     clientId,
			ClientSecret: clientSecret,
		},
	}
}

func (s *Facebook) Name() string {
	return "facebook"
}

func (s *Facebook) GetClient(token string) Client {
	return NewTokenClient(s, token)
}

func (s *Facebook) GetExchangeUrl(token string) string {
	exchangeUrl := s.TokenUrl + "?"

	vals := url.Values{}
	vals.Add("grant_type", "fb_exchange_token")
	vals.Add("client_id", s.GetClientID())
	vals.Add("client_secret", s.GetClientSecrect())
	vals.Add("fb_exchange_token", token)

	exchangeUrl += vals.Encode()

	return exchangeUrl
}

func (s *Facebook) Exchange(token string) (string, apperror.Error) {
	data, err := exchangeToken(s.GetExchangeUrl(token))
	if err != nil {
		return "", err
	}

	vals, err2 := url.ParseQuery(data)
	if err2 != nil {
		return "", apperror.Wrap(err, "body_parse_error", "")
	}

	appToken := vals.Get("access_token")
	if appToken == "" {
		return "", apperror.New("no_access_token_in_response")
	}

	return appToken, nil
}

func (s *Facebook) GetUserData(token string) (*UserData, apperror.Error) {
	c := s.GetClient(token)
	_, data, err := c.Do("GET", "/me", nil)
	if err != nil {
		return nil, err
	}

	userData := &UserData{
		ID:   data["id"].(string),
		Data: data,
	}

	// Email  is only available if "email" scope was requested.
	if email, ok := data["email"].(string); ok {
		userData.Email = email
	}

	return userData, nil
}
