package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
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

type AuthAdaptorPassword struct {}

func (a AuthAdaptorPassword) GetName() string {
	return "password"
}

func (a AuthAdaptorPassword) BuildData(rawData interface{}) (interface{}, error) {
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

func (a AuthAdaptorPassword) Authenticate(rawData, rawCheckable interface{}) (bool, error) {
	hash, _ := GetStringFromMap(rawData, "hash")
	pw, _ := GetStringFromMap(rawCheckable, "password")

	if hash == "" || pw == "" {
		return false, errors.New("invalid_data")
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw))
	return err == nil, err
}
