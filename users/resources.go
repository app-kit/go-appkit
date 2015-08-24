package users

import(
	"crypto/rand"
	"math/big"	
	"time"

	"github.com/theduke/appkit"
)

func randomToken() string {
	n := 32

	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	symbols := big.NewInt(int64(len(alphanum)))
	states := big.NewInt(0)
	states.Exp(symbols, big.NewInt(int64(n)), nil)
	r, err := rand.Int(rand.Reader, states)
	if err != nil {
		return ""
	}

	var bytes = make([]byte, n)
	r2 := big.NewInt(0)
	symbol := big.NewInt(0)
	for i := range bytes {
		r2.DivMod(r, symbols, symbol)
		r, r2 = r2, r
		bytes[i] = alphanum[symbol.Int64()]
	}
	return string(bytes)
}

type SessionResource struct {
	appkit.Resource
	UserHandler UserHandler
}

func (res SessionResource) StartSession(user appkit.ApiUser) (appkit.ApiSession, appkit.ApiError) {
	token := randomToken()
	if token == "" {
		return nil, appkit.Error{Code: "token_creation_failed"}
	}

	rawSession, err := res.Backend.GetType(res.Model.GetName())
	if err != nil  {
		return nil, err
	}

	// TOdo: fix this.
	session := rawSession.(appkit.ApiSession)

	session.SetUserID(user.GetID())
	session.SetToken(token)
	session.SetStartedAt(time.Now())
	session.SetValidUntil(time.Now().Add(time.Hour * 12))

	err = res.Create(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (res SessionResource) ApiCreate(obj appkit.ApiModel, r appkit.ApiRequest) appkit.ApiResponse {
	meta := r.GetMeta()

	userIdentifier := meta.GetString("user")
	if userIdentifier == "" {
		return appkit.NewErrorResponse("user_missing", "Expected 'user' in metadata.")
	}

	// Find user.
	userResource := res.UserHandler.GetUserResource()

	rawUser, err := userResource.FindOneBy(map[string]interface{}{"email": userIdentifier})
	if err != nil {
		return appkit.Response{Error: err}
	}
	if rawUser == nil {
		rawUser, err = userResource.FindOneBy(map[string]interface{}{"username": userIdentifier})
		if err != nil {
			return appkit.Response{Error: err}
		}
		if rawUser == nil {
			return appkit.NewErrorResponse("user_not_found", "User not found for identifier: " + userIdentifier)
		}
	}
	user := rawUser.(appkit.ApiUser)

	adaptor := meta.GetString("adaptor")
	if adaptor == "" {
		return appkit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.")
	}

	data, ok := meta.Get("auth-data")
	if !ok {
		return appkit.NewErrorResponse("auth_data_missing", "Expected 'auth-data' in metadata.")
	}

	err = res.UserHandler.AuthenticateUser(user, adaptor, data)
	if err != nil {
		return appkit.Response{Error: err}
	}

	session, err := res.StartSession(user)
	if err != nil {
		return appkit.Response{Error: err}
	}

	return appkit.Response{
  	Data: session,
  }
}

/**
 * User resource.
 */

 type UserResource struct {
 	appkit.Resource
 	UserHandler UserHandler
 }

 func (res UserResource) Create(obj appkit.ApiModel) appkit.ApiError {
 	panic("Must use UserResource.CreateUser()")
 }

func (res UserResource) CreateUser(user appkit.ApiUser, adaptorName string, adaptorData interface{}) appkit.ApiError {
	obj := user.(appkit.ApiModel)

	if res.BeforeCreate != nil {
		if err := res.BeforeCreate(obj); err != nil {
			return err
		}
	}

	if err := res.UserHandler.CreateUser(user, adaptorName, adaptorData); err != nil {
		return err
	}

	if res.AfterCreate != nil {
		if err := res.AfterCreate(obj); err != nil {
			return err
		}
	}

	return nil
}

func (res UserResource) ApiCreate(obj appkit.ApiModel, r appkit.ApiRequest) appkit.ApiResponse {
 	meta := r.GetMeta()

 	adaptor := meta.GetString("adaptor")
	if adaptor == "" {
		return appkit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.")
	}

	data, ok := meta.Get("auth-data")
	if !ok {
		return appkit.NewErrorResponse("auth_data_missing", "Expected 'auth-data' in metadata.")
	}

	err := res.CreateUser(obj.(appkit.ApiUser), adaptor, data)
	if err != nil {
		return appkit.Response{Error: err}
	}

	return appkit.Response{
  	Data: obj,
  }
}
