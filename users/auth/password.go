package auth

import (
	"golang.org/x/crypto/bcrypt"

	kit "github.com/theduke/go-appkit"
	. "github.com/theduke/go-appkit/error"
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

type AuthAdaptorPassword struct{}

func (a AuthAdaptorPassword) GetName() string {
	return "password"
}

func (a AuthAdaptorPassword) BuildData(user kit.User, rawData interface{}) (interface{}, Error) {
	pw, _ := GetStringFromMap(rawData, "password")
	if pw == "" {
		return nil, AppError{Code: "invalid_data"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	if err != nil {
		return nil, AppError{
			Code:    "hash_error",
			Message: err.Error(),
		}
	}

	data := map[string]interface{}{
		"hash": string(hash),
	}

	return interface{}(data), nil
}

func (a AuthAdaptorPassword) Authenticate(user kit.User, rawData, rawCheckable interface{}) (bool, Error) {
	hash, _ := GetStringFromMap(rawData, "hash")
	pw, _ := GetStringFromMap(rawCheckable, "password")

	if hash == "" || pw == "" {
		return false, AppError{Code: "invalid_data"}
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	return err == nil, nil
}
