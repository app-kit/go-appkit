package users

import(
	"errors"
	"crypto/rand"
	"math/big"	
	"time"
	//"log"

	"github.com/theduke/appkit/api"
)

type Data map[string]interface{}


func NewUserHandler() *BaseUserHandler {
	p := BaseUserHandler{
		AuthAdaptors: make(map[string]AuthAdaptor),
	}


	pwAdaptor := AuthAdaptorPassword{}
	p.AddAuthAdaptor(pwAdaptor)

	return &p
}

type BaseUserHandler struct	{
	Sessions SessionProvider
	Users UserProvider
	Auth AuthProvider

	AuthAdaptors map[string]AuthAdaptor

	UserProfileBuilder func(User) error
}

func (p *BaseUserHandler) SetSessionProvider(s SessionProvider) {
	p.Sessions = s
}

func (h *BaseUserHandler) SetUserProvider(p UserProvider) {
	h.Users = p
}

func (h *BaseUserHandler) SetAuthProvider(p AuthProvider) {
	h.Auth = p
}

func (p *BaseUserHandler) GetAuthAdaptor(name string) AuthAdaptor {
	return p.AuthAdaptors[name];
}

func (p *BaseUserHandler) AddAuthAdaptor(a AuthAdaptor) {
	p.AuthAdaptors[a.GetName()] = a
}

func (p *BaseUserHandler) GetUser(id string) (User, error) {
	return p.Users.FindOne(id)
}

func (p *BaseUserHandler) AuthenticateUser(userId string, authAdaptorName string, authData interface{}) (User, error) {
	user, err := p.Users.FindOne(userId)
	if err != nil {
		return nil, api.ApiError{
			Code: "user_not_found", 
			Message: err.Error(),
		}
	}

	if !user.IsActive() {
		return nil, api.ApiError{Code: "user_inactive"}
	}

	authAdaptor := p.GetAuthAdaptor(authAdaptorName)
	if authAdaptor == nil {
		return nil, api.ApiError{
			Code: "unknown_auth_adaptor", 
			Message: "Unknown auth adaptor: " + authAdaptorName}
	}

	auth, err := p.Auth.FindOne(authAdaptorName, user)
	if err != nil {
		return nil, api.ApiError{Code: "auth_error", Message: err.Error()}
	}

	cleanData, err := auth.GetData()
	if err != nil {
		return nil, api.ApiError{
			Code: "invalid_auth_data", 
			Message: err.Error(),
		}
	}

	ok, err := authAdaptor.Authenticate(cleanData, authData)
	if err != nil {
		return nil, api.ApiError{Code: "auth_error", Message: err.Error()}
	}
	if !ok {
		return nil, api.ApiError{Code: "invalid_credentials"}
	}

	return user, nil
}

func (p *BaseUserHandler) BuildRandomToken() string {
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

func (p *BaseUserHandler) StartSession(user *User, authProviderName string, authData interface{}) (*User, *Session, error) {
	token := p.BuildRandomToken()
	if token == "" {
		return nil, nil, errors.New("token_creation_failed")
	}

	session := p.Sessions.NewSession()
	session.SetUserID((*user).GetID())
	session.SetToken(token)
	session.SetStartedAt(time.Now())
	session.SetValidUntil(time.Now().Add(time.Hour * 12))

	err := p.Sessions.Create(session)
	if err != nil {
		return nil, nil, err
	}

	return user, &session, nil	
}

func (p *BaseUserHandler) VerifySession(token string) (Session, error) {
	session, err := p.Sessions.FindOne(token)
	if err != nil {
		return nil, err
	}

	if session.GetValidUntil().Sub(time.Now()) < 1 {
		return nil, errors.New("session_expired")
	}

	// Prolong session
	session.SetValidUntil(time.Now().Add(time.Hour * 12))
	if err := p.Sessions.Update(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (p *BaseUserHandler) EndSession(session Session) error {
	return p.Sessions.Delete(session)
}

func (p *BaseUserHandler) CreateUser(user User, adaptorName string, authData interface{}) error {
	adaptor := p.GetAuthAdaptor(adaptorName)
	if adaptor == nil  {
		return errors.New("unknown_auth_adaptor")
	}

	data, err := adaptor.BuildData(authData)
	if err != nil {
		return api.ApiError{Code: "adaptor_error", Message: err.Error()}
	}

	if user.GetUsername() == "" {
		user.SetUsername(user.GetEmail())
	}
	
	// Check if user with same username or email exists.
	if u, err := p.Users.FindByEmail(user.GetEmail()); err != nil {
		return err
	} else if u != nil {
		return errors.New("email_exists")
	}

	if u, err := p.Users.FindByUsername(user.GetUsername()); err != nil {
		return err
	} else if u != nil {
		return errors.New("username_exists")
	}

	if err := p.Users.Create(user); err != nil {
		return err
	}

	auth := p.Auth.NewItem(user.GetID(), adaptorName, data)
	if err := p.Auth.Create(auth); err != nil {
		p.Users.Delete(user)
		return api.ApiError{Code: "auth_save_failed", Message: err.Error()}
	}

	return nil
}

func (p *BaseUserHandler) DeleteUser(user User) error {
	auths, err := p.Auth.FindByUser(user)
	if err != nil {
		return err
	}

	for _, auth := range auths {
		if err := p.Auth.Delete(auth); err != nil {
			return err
		}
	}

	// Now delete user.
	if err := p.Users.Delete(user); err != nil {
		return err
	}

	return nil
}
