package password

import (
	"fmt"

	db "github.com/theduke/go-dukedb"
	"golang.org/x/crypto/bcrypt"

	kit "github.com/theduke/go-appkit"
)

func GetStringFromMap(data map[string]interface{}, field string) (string, bool) {
	if data == nil {
		return "", false
	}

	pw, ok := data[field].(string)
	if !ok {
		return "", false
	}

	return pw, true
}

type AuthItemPassword struct {
	// ID serves as UserID.
	db.BaseStrIDModel

	Hash string `db:"not-null;max:150"`
}

// Ensure AuthItemPassword implements kit.AuthItem.
var _ kit.AuthItem = (*AuthItemPassword)(nil)

func (item *AuthItemPassword) Collection() string {
	return "users_auth_passwords"
}

func (item *AuthItemPassword) GetUserID() interface{} {
	return item.ID
}

func (item *AuthItemPassword) SetUserID(id interface{}) error {
	return item.SetID(id)
}

func (item *AuthItemPassword) GetUser() kit.User {
	return nil
}

func (item *AuthItemPassword) SetUser(u kit.User) {
	item.SetUserID(u.GetID())
}

type AuthAdaptorPassword struct {
	backend db.Backend
}

// Ensure AuthAdaptorPassword implements appkit.AuthAdaptor.
var _ kit.AuthAdaptor = (*AuthAdaptorPassword)(nil)

func (a AuthAdaptorPassword) Name() string {
	return "password"
}

func (a *AuthAdaptorPassword) Backend() db.Backend {
	return a.backend
}

func (a *AuthAdaptorPassword) SetBackend(b db.Backend) {
	b.RegisterModel(&AuthItemPassword{})
	a.backend = b
}

func (a *AuthAdaptorPassword) RegisterUser(user kit.User, data map[string]interface{}) (kit.AuthItem, kit.Error) {
	if data == nil {
		return nil, kit.AppError{Code: "invalid_nil_data"}
	}
	pw, _ := GetStringFromMap(data, "password")
	if pw == "" {
		return nil, kit.AppError{Code: "invalid_data_no_password"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	if err != nil {
		return nil, kit.WrapError(err, "hash_errr", "")
	}

	item := &AuthItemPassword{
		Hash: string(hash),
	}
	item.SetID(user.GetID())

	return item, nil
}

func (a *AuthAdaptorPassword) GetItem(userId string) (*AuthItemPassword, kit.Error) {
	rawItem, err := a.backend.FindOne("users_auth_passwords", userId)
	if err != nil {
		return nil, err
	} else if rawItem == nil {
		return nil, kit.AppError{
			Code:    "no_authitem",
			Message: fmt.Sprintf("No password auth item could be found for userID %v", userId),
		}
	}

	return rawItem.(*AuthItemPassword), nil
}

func (a AuthAdaptorPassword) Authenticate(userID string, data map[string]interface{}) (string, kit.Error) {
	if userID == "" {
		return "", kit.AppError{Code: "empty_user_id"}
	}

	pw, _ := GetStringFromMap(data, "password")
	if pw == "" {
		return "", kit.AppError{Code: "invalid_data_no_password"}
	}

	item, err := a.GetItem(userID)
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(item.Hash), []byte(pw)); err != nil {
		return "", kit.AppError{Code: "invalid_credentials"}
	}

	return userID, nil
}

func (a *AuthAdaptorPassword) ChangePassword(userId, newPw string) kit.Error {
	item, err := a.GetItem(userId)
	if err != nil {
		return err
	}

	hash, err2 := bcrypt.GenerateFromPassword([]byte(newPw), 10)
	if err2 != nil {
		return kit.WrapError(err, "hash_err", "")
	}

	item.Hash = string(hash)
	if err := a.backend.Update(item); err != nil {
		return err
	}

	return nil
}
