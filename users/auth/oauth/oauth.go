package oauth

import (
	"errors"

	"golang.org/x/oauth2"

	"github.com/theduke/go-appkit/users"
)

func GetStringFromMap(rawData interface{}, field string) (string, bool) {
	data, ok := rawData.(map[string]interface{})
	if !ok {
		return "", false
	}

	pw, ok := data[field].(string)
	if !ok {
		return "", false
	}

	return pw, true
}

type Config struct {
	ClientID string
	ClientSecret string
	AuthUrl string
	RedirectUrl string
	TokenUrl string
}

type AuthAdaptorOauth struct{
	endpoints map[string]*Config
}

// Ensure AuthAdaptorOauth implements ApiAuthAdaptor.
var _ users.ApiAuthAdaptor = (*AuthAdaptorOauth)(nil)


func New() *AuthAdaptorOauth {
	return &AuthAdaptorOauth {
		endpoints: make(map[string]*Config),
	}
}

func (a AuthAdaptorOauth) GetName() string {
	return "password"
}

func (a *AuthAdaptorOauth) RegisterEndpoint(name string, conf *Config) {
	a.endpoints[name] = conf
}

func (a AuthAdaptorOauth) BuildData(user users.ApiUser, rawData interface{}) (interface{}, error) {
	pw, _ := GetStringFromMap(rawData, "password")
	if pw == "" {
		return nil, errors.New("invalid_data")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"hash": string(hash),
	}

	return interface{}(data), nil
}

func (a AuthAdaptorOauth) Authenticate(user users.ApiUser, rawData, rawCheckable interface{}) (bool, error) {
	hash, _ := GetStringFromMap(rawData, "hash")
	pw, _ := GetStringFromMap(rawCheckable, "password")

	if hash == "" || pw == "" {
		return false, errors.New("invalid_data")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	return err == nil, err
}
