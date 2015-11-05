package oauth

import (
	"fmt"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"
	"github.com/theduke/go-reflector"

	kit "github.com/app-kit/go-appkit"
	//"github.com/app-kit/go-appkit/users"
)

type AuthItemOauth struct {
	db.StrIdModel
	Service        string `db:"required;max:100;"`
	UserId         string `db:"required;max:150;"`
	ExternalUserId string `db:"required;max:100;"`
	Token          string `db:"required;max:500;"`
}

// Ensure AuthItemOauth implements kit.AuthItem and kit.UserModel
var _ kit.AuthItem = (*AuthItemOauth)(nil)
var _ kit.UserModel = (*AuthItemOauth)(nil)

func (item *AuthItemOauth) Collection() string {
	return "users_auth_oauth"
}

// Implement kit.UserModel interface.

func (item *AuthItemOauth) GetUserId() interface{} {
	return item.UserId
}

func (item *AuthItemOauth) SetUserId(id interface{}) error {
	convertedId, err := reflector.R(id).ConvertTo("")
	if err != nil {
		return err
	}
	item.UserId = convertedId.(string)
	return nil
}

func (item *AuthItemOauth) GetUser() kit.User {
	return nil
}

func (item *AuthItemOauth) SetUser(u kit.User) {
	item.SetUserId(u.GetId())
}

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

type UserData struct {
	Email    string
	Username string
	Id       string
	Data     map[string]interface{}
}

type AuthAdaptorOauth struct {
	backend  db.Backend
	services map[string]Service
}

// Ensure AuthAdaptorOauth implements appkit.AuthAdaptor.
var _ kit.AuthAdaptor = (*AuthAdaptorOauth)(nil)

func NewAdaptor() *AuthAdaptorOauth {
	return &AuthAdaptorOauth{
		services: make(map[string]Service),
	}
}

func (item *AuthAdaptorOauth) Backend() db.Backend {
	return item.backend
}

func (item *AuthAdaptorOauth) SetBackend(b db.Backend) {
	b.RegisterModel(&AuthItemOauth{})
	item.backend = b
}

func (a AuthAdaptorOauth) Name() string {
	return "oauth"
}

func (a *AuthAdaptorOauth) RegisterService(service Service) {
	a.services[service.Name()] = service
}

func (a *AuthAdaptorOauth) RegisterUser(user kit.User, data map[string]interface{}) (kit.AuthItem, apperror.Error) {
	serviceName, _ := GetStringFromMap(data, "service")
	if serviceName == "" {
		return nil, apperror.New("invalid_data_missing_service")
	}

	service := a.services[serviceName]
	if service == nil {
		return nil, &apperror.Err{
			Code:    "unconfigured_service",
			Message: fmt.Sprintf("The oauth service '%v' was not configured in oauth auth adaptor", serviceName),
		}
	}

	accessToken, _ := GetStringFromMap(data, "access_token")
	if accessToken == "" {
		return nil, apperror.New("invalid_data_missing_access_token")
	}

	// Exchange access token for long lived token.
	// This also verifies that the supplied token is valid.
	appToken, err := service.Exchange(accessToken)
	if err != nil {
		return nil, apperror.Wrap(err, "oauth_exchange_token_error", "")
	}

	userData, err := service.GetUserData(appToken)
	if err != nil {
		return nil, apperror.Wrap(err, "fetch_user_data_failed", "")
	}

	if userData.Id == "" {
		return nil, &apperror.Err{
			Code:    "fetched_userdata_missing_user_id",
			Message: "The userData fetched from the service does not contain a userId",
		}
	}

	item := &AuthItemOauth{
		Service:        serviceName,
		UserId:         user.GetStrId(),
		ExternalUserId: userData.Id,
		Token:          appToken,
	}
	item.Id = serviceName + "_" + userData.Id

	// Fill in user information.

	if user.GetEmail() == "" {
		if userData.Email != "" {
			user.SetEmail(userData.Email)
			user.SetIsEmailConfirmed(true)
		} else {
			return nil, &apperror.Err{
				Code:    "oauth_service_insufficient_data_error",
				Message: fmt.Sprintf("The oauth service %v did not supply the users email, which is required", serviceName),
			}
		}
	}

	if user.GetUsername() == "" && userData.Username != "" {
		user.SetUsername(userData.Username)
	}

	return item, nil
}

func (a AuthAdaptorOauth) Authenticate(_ string, data map[string]interface{}) (string, apperror.Error) {
	serviceName, _ := GetStringFromMap(data, "service")
	if serviceName == "" {
		return "", apperror.New("invalid_data_missing_service")
	}

	service := a.services[serviceName]
	if service == nil {
		return "", &apperror.Err{
			Code:    "unconfigured_service",
			Message: fmt.Sprintf("The oauth service '%v' was not configured in oauth auth adaptor", serviceName),
		}
	}

	accessToken, _ := GetStringFromMap(data, "access_token")
	if accessToken == "" {
		return "", apperror.New("invalid_data_missing_access_token")
	}

	// Exchange access token for long lived token.
	// This also verifies that the supplied token is valid.
	appToken, err := service.Exchange(accessToken)
	if err != nil {
		return "", apperror.Wrap(err, "oauth_exchange_token_error", "")
	}

	userData, err := service.GetUserData(appToken)
	if err != nil {
		return "", apperror.Wrap(err, "fetch_user_data_failed", "")
	}

	if userData.Id == "" {
		return "", &apperror.Err{
			Code:    "fetched_userdata_missing_user_id",
			Message: "The userData fetched from the service does not contain a userId",
		}
	}

	// Find the auth item.
	rawItem, err := a.backend.Q("users_auth_oauth").Filter("service", serviceName).Filter("external_user_id", userData.Id).First()
	if err != nil {
		return "", apperror.Wrap(err, "auth_item_query_error", "")
	} else if rawItem == nil {
		return "", &apperror.Err{
			Code:    "no_authitem",
			Message: fmt.Sprintf("No oauth auth item could be found for userId %v", userData.Id),
		}
	}

	item := rawItem.(*AuthItemOauth)

	return item.UserId, nil
}
