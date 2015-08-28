package users

import(
	"crypto/rand"
	"math/big"	
	"time"

	kit "github.com/theduke/appkit"
	db "github.com/theduke/dukedb"
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

type SessionResourceHooks struct {
	ApiUpdateAllowed bool
	ApiDeleteAllowed bool
}

func StartSession(res kit.ApiResource, user kit.ApiUser) (kit.ApiSession, kit.ApiError) {
	token := randomToken()
	if token == "" {
		return nil, kit.Error{Code: "token_creation_failed"}
	}

	rawSession, err := res.GetBackend().NewModel(res.GetModel().GetCollection())
	if err != nil  {
		return nil, err
	}
	session := rawSession.(kit.ApiSession)

	session.SetUserID(user.GetID())
	session.SetToken(token)
	session.SetStartedAt(time.Now())
	session.SetValidUntil(time.Now().Add(time.Hour * 12))

	err = res.Create(session, nil	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (hooks SessionResourceHooks) ApiCreate(res kit.ApiResource, obj db.Model, r kit.ApiRequest) kit.ApiResponse {
	meta := r.GetMeta()

	userIdentifier := meta.GetString("user")
	if userIdentifier == "" {
		return kit.NewErrorResponse("user_missing", "Expected 'user' in metadata.")
	}

	// Find user.
	userResource := res.GetUserHandler().GetUserResource()

	rawUser, err := userResource.Q().
	  Filter("username", userIdentifier).Or("email", userIdentifier).First()

	if err != nil {
		return &kit.Response{Error: err}
	} else if rawUser == nil {
		return kit.NewErrorResponse("user_not_found", "User not found for identifier: " + userIdentifier)
	}

	user := rawUser.(kit.ApiUser)

	adaptor := meta.GetString("adaptor")
	if adaptor == "" {
		return kit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.")
	}

	data, ok := meta.Get("auth-data")
	if !ok {
		kit.NewErrorResponse("auth_data_missing", "Expected 'auth-data' in metadata.")
	}

	err = res.GetUserHandler().AuthenticateUser(user, adaptor, data)
	if err != nil {
		return &kit.Response{Error: err}
	}

	session, err := StartSession(res, user)
	if err != nil {
		return &kit.Response{Error: err}
	}
		
	return &kit.Response{
		Data: session,
	}
}


/**
 * User resource.
 */

type UserResourceHooks struct {
	ProfileModel kit.ApiUserProfile
}

func (hooks UserResourceHooks) ApiCreate(res kit.ApiResource, obj db.Model, r kit.ApiRequest) kit.ApiResponse {
	meta := r.GetMeta()

 	adaptor := meta.GetString("adaptor")
	if adaptor == "" {
		return kit.NewErrorResponse("adaptor_missing", "Expected 'adaptor' in metadata.")
	}

	data, ok := meta.Get("auth-data")
	if !ok {
		return kit.NewErrorResponse("auth_data_missing", "Expected 'auth-data' in metadata.")
	}

	user := obj.(kit.ApiUser)
	if err := res.GetUserHandler().CreateUser(user, adaptor, data); err != nil {
		return &kit.Response{Error: err}
	}

	return &kit.Response{
		Data: user,
	}
}

func (hooks UserResourceHooks) AllowFind(res kit.ApiResource, obj db.Model, user kit.ApiUser) bool {
	u := obj.(kit.ApiUser)
	return u.GetID() == user.GetID()
}

func (hooks UserResourceHooks) AllowUpdate(res kit.ApiResource, obj db.Model, old db.Model, user kit.ApiUser) bool {
	return user != nil && obj.GetID() == user.GetID()
}

func (hooks UserResourceHooks) AllowDelete(res kit.ApiResource, obj db.Model, old db.Model, user kit.ApiUser) bool {
	return false
}

type RoleResourceHooks struct {

}

type PermissionResourceHooks struct {
	
}