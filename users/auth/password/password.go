package password

import (
	"fmt"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"
	"golang.org/x/crypto/bcrypt"

	kit "github.com/app-kit/go-appkit"
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
	// Id serves as UserId.
	db.StrIdModel

	Hash string `db:"required;max:150"`
}

// Ensure AuthItemPassword implements kit.AuthItem.
var _ kit.AuthItem = (*AuthItemPassword)(nil)

func (item *AuthItemPassword) Collection() string {
	return "users_auth_passwords"
}

func (item *AuthItemPassword) GetUserId() interface{} {
	return item.Id
}

func (item *AuthItemPassword) SetUserId(id interface{}) error {
	return item.SetId(id)
}

func (item *AuthItemPassword) GetUser() kit.User {
	return nil
}

func (item *AuthItemPassword) SetUser(u kit.User) {
	item.SetUserId(u.GetId())
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

func (a *AuthAdaptorPassword) RegisterUser(user kit.User, data map[string]interface{}) (kit.AuthItem, apperror.Error) {
	if data == nil {
		return nil, apperror.New("invalid_nil_data")
	}
	pw, _ := GetStringFromMap(data, "password")
	if pw == "" {
		return nil, apperror.New("invalid_data_no_password")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), 10)
	if err != nil {
		return nil, apperror.Wrap(err, "hash_errr", "")
	}

	item := &AuthItemPassword{
		Hash: string(hash),
	}
	item.SetId(user.GetId())

	return item, nil
}

func (a *AuthAdaptorPassword) GetItem(userId string) (*AuthItemPassword, apperror.Error) {
	rawItem, err := a.backend.FindOne("users_auth_passwords", userId)
	if err != nil {
		return nil, err
	} else if rawItem == nil {
		return nil, &apperror.Err{
			Code:    "no_authitem",
			Message: fmt.Sprintf("No password auth item could be found for userId %v", userId),
		}
	}

	return rawItem.(*AuthItemPassword), nil
}

func (a AuthAdaptorPassword) Authenticate(userId string, data map[string]interface{}) (string, apperror.Error) {
	if userId == "" {
		return "", apperror.New("empty_user_id")
	}

	pw, _ := GetStringFromMap(data, "password")
	if pw == "" {
		return "", apperror.New("invalid_data_no_password", true)
	}

	item, err := a.GetItem(userId)
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(item.Hash), []byte(pw)); err != nil {
		return "", apperror.New("invalid_credentials", true)
	}

	return userId, nil
}

func (a *AuthAdaptorPassword) ChangePassword(userId, newPw string) apperror.Error {
	if newPw == "" {
		return apperror.New("empty_password", "The password may not be empty", true)
	}

	item, err := a.GetItem(userId)
	if err != nil {
		return err
	}

	hash, err2 := bcrypt.GenerateFromPassword([]byte(newPw), 10)
	if err2 != nil {
		return apperror.Wrap(err, "hash_err", "")
	}

	item.Hash = string(hash)
	if err := a.backend.Update(item); err != nil {
		return err
	}

	return nil
}
