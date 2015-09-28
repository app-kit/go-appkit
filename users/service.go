package users

import (
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"
	"github.com/twinj/uuid"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/resources"
	"github.com/theduke/go-appkit/users/auth/oauth"
	"github.com/theduke/go-appkit/users/auth/password"
)

type Service struct {
	debug bool
	deps  kit.Dependencies

	backend db.Backend

	Users    kit.Resource
	Profiles kit.Resource
	Sessions kit.Resource
	Tokens   kit.Resource

	Roles       kit.Resource
	Permissions kit.Resource

	AuthAdaptors map[string]kit.AuthAdaptor
}

// Ensure UserService implements kit.UserService.
var _ kit.UserService = (*Service)(nil)

func NewService(deps kit.Dependencies, backend db.Backend, profileModel kit.UserProfile) *Service {
	h := Service{
		deps: deps,
	}

	h.AuthAdaptors = make(map[string]kit.AuthAdaptor)

	// Register auth adaptors.
	h.AddAuthAdaptor(&password.AuthAdaptorPassword{})
	h.AddAuthAdaptor(oauth.NewAdaptor())

	// Build resources.
	var userModel kit.Model
	if backend.HasStringIDs() {
		userModel = &UserStrID{}
	} else {
		userModel = &UserIntID{}
	}
	users := resources.NewResource(userModel, UserResourceHooks{}, true)
	h.Users = users

	if profileModel != nil {
		profiles := resources.NewResource(profileModel, nil, false)
		h.Profiles = profiles
	}

	var sessionModel kit.Model
	if backend.HasStringIDs() {
		sessionModel = &Session{}
	} else {
		sessionModel = &IntUserSession{}
	}
	sessions := resources.NewResource(sessionModel, SessionResourceHooks{}, true)
	h.Sessions = sessions

	h.Tokens = resources.NewResource(&Token{}, nil, false)

	roles := resources.NewResource(&Role{}, RoleResourceHooks{}, true)
	h.Roles = roles

	permissions := resources.NewResource(&Permission{}, PermissionResourceHooks{}, true)
	h.Permissions = permissions

	// Ensure proper backend setup.
	h.SetBackend(backend)

	return &h
}

func (s *Service) Debug() bool {
	return s.debug
}

func (s *Service) SetDebug(x bool) {
	s.debug = x
}

func (s *Service) Dependencies() kit.Dependencies {
	return s.deps
}

func (s *Service) SetDependencies(x kit.Dependencies) {
	s.deps = x
	if s.backend == nil && x.DefaultBackend() != nil {
		s.SetBackend(x.DefaultBackend())
	}
}

func (s *Service) Backend() db.Backend {
	return s.backend
}

func (s *Service) SetBackend(b db.Backend) {
	s.Users.SetBackend(b)
	b.RegisterModel(s.Users.Model())

	if s.Profiles != nil {
		s.Profiles.SetBackend(b)
		b.RegisterModel(s.Profiles.Model())
	}

	s.Sessions.SetBackend(b)
	b.RegisterModel(s.Sessions.Model())

	s.Tokens.SetBackend(b)
	b.RegisterModel(s.Tokens.Model())

	s.Roles.SetBackend(b)
	b.RegisterModel(s.Roles.Model())

	s.Permissions.SetBackend(b)
	b.RegisterModel(s.Permissions.Model())

	for name := range s.AuthAdaptors {
		s.AuthAdaptors[name].SetBackend(b)
	}

	s.backend = b
}

func (h *Service) AuthAdaptor(name string) kit.AuthAdaptor {
	return h.AuthAdaptors[name]
}

func (h *Service) AddAuthAdaptor(a kit.AuthAdaptor) {
	h.AuthAdaptors[a.Name()] = a
}

func (h *Service) UserResource() kit.Resource {
	return h.Users
}

func (h *Service) SetUserResource(x kit.Resource) {
	h.Users = x
}

func (h *Service) ProfileResource() kit.Resource {
	return h.Profiles
}

func (h *Service) SetProfileResource(x kit.Resource) {
	h.Profiles = x
}

func (h *Service) SessionResource() kit.Resource {
	return h.Sessions
}

func (h *Service) SetSessionResource(x kit.Resource) {
	h.Sessions = x
}

func (h *Service) TokenResource() kit.Resource {
	return h.Tokens
}

func (h *Service) SetTokenResource(x kit.Resource) {
	h.Tokens = x
}

func (h *Service) ProfileModel() kit.UserProfile {
	return h.Profiles.Model().(kit.UserProfile)
}

/**
 * RBAC resources.
 */

func (u *Service) RoleResource() kit.Resource {
	return u.Roles
}

func (u *Service) SetRoleResource(x kit.Resource) {
	u.Roles = x
}

func (u *Service) PermissionResource() kit.Resource {
	return u.Permissions
}

func (u *Service) SetPermissionResource(x kit.Resource) {
	u.Permissions = x
}

func (s *Service) BuildToken(typ, userId string, expiresAt time.Time) (kit.UserToken, apperror.Error) {
	token := uuid.NewV4().String()

	tokenItem := s.Tokens.CreateModel().(kit.UserToken)
	tokenItem.SetType(typ)
	tokenItem.SetToken(token)
	tokenItem.SetUserID(userId)

	if err := s.Tokens.Create(tokenItem, nil); err != nil {
		return nil, apperror.Wrap(err, "token_create_error", "Could not save token to database")
	}

	return tokenItem, nil
}

func (s *Service) FindUser(userId interface{}) (kit.User, apperror.Error) {
	rawUser, err := s.Users.Q().Filter("id", userId).Join("Roles.Permissions").First()
	if err != nil {
		return nil, err
	} else if rawUser == nil {
		return nil, nil
	}

	user := rawUser.(kit.User)

	if s.Profiles != nil {
		profile, err := s.Users.Backend().Q(s.Profiles.Collection()).Filter("id", user.GetID()).First()
		if err != nil {
			return nil, apperror.Wrap(err, "profile_query_error")
		} else if profile != nil {
			user.SetProfile(profile.(kit.UserProfile))
		}
	}

	return user, nil
}

func (s *Service) CreateUser(user kit.User, adaptorName string, authData map[string]interface{}) apperror.Error {
	adaptor := s.AuthAdaptor(adaptorName)
	if adaptor == nil {
		return &apperror.Err{
			Code:    "unknown_auth_adaptor",
			Message: fmt.Sprintf("Auth adaptor %v was not registered with user service", adaptorName),
		}
	}

	authItem, err := adaptor.RegisterUser(user, authData)
	if err != nil {
		return apperror.Wrap(err, "adaptor_error", "")
	}

	if user.GetUsername() == "" {
		user.SetUsername(user.GetEmail())
	}

	// Check if user with same username or email exists.
	oldUser, err2 := s.Users.Q().
		Filter("email", user.GetEmail()).Or("username", user.GetUsername()).First()
	if err2 != nil {
		return err2
	} else if oldUser != nil {
		return &apperror.Err{
			Code:    "user_exists",
			Message: "A user with the username or email already exists",
		}
	}

	user.SetIsActive(true)

	profile := user.GetProfile()

	// If a profile is configured, and the user does not have a profile yet,
	// create a new one.
	if s.Profiles != nil && profile == nil {
		profile = s.Profiles.CreateModel().(kit.UserProfile)
		user.SetProfile(profile)
	}

	if err := s.Users.Create(user, nil); err != nil {
		return err
	}

	// Create profile if one exists.

	if profile != nil {
		profile.SetUser(user)
		if err := s.Profiles.Create(profile, nil); err != nil {
			s.Users.Backend().Delete(user)
			return apperror.Wrap(err, "user_profile_create_error", "Could not create the user profile")
		}
	}

	// Persist auth item.
	if authItemUserId, ok := authItem.(kit.UserModel); ok {
		authItemUserId.SetUserID(user.GetID())
	}
	if err := s.Users.Backend().Create(authItem); err != nil {
		s.Users.Backend().Delete(user)
		if profile != nil {
			s.Profiles.Backend().Delete(profile)
		}
		return apperror.Wrap(err, "auth_item_create_error", "")
	}

	if err := s.SendConfirmationEmail(user); err != nil {
		s.deps.Logger().Errorf("Could not send confirmation email: %v", err)
	}

	return nil
}

func (s *Service) SendConfirmationEmail(user kit.User) apperror.Error {
	// Check that an email service is configured.

	mailService := s.deps.EmailService()
	if mailService == nil {
		return apperror.New("no_email_service")
	}

	conf := s.deps.Config()

	// Check that sending is enabled.
	if !conf.UBool("users.sendEmailConfirmationEmail", true) {
		return nil
	}

	// Generate a token.
	tokenItem, err := s.BuildToken("email_confirmation", user.GetStrID(), time.Time{})
	if err != nil {
		return err
	}
	token := tokenItem.GetToken()

	// Build the confirmation url.

	confirmationPath := conf.UString("users.emailConfirmationPath")
	if confirmationPath == "" {
		return &apperror.Err{
			Code:    "no_email_confirmation_path",
			Message: "Config must specify users.emailConfirmationPath",
		}
	}

	if !strings.Contains(confirmationPath, "{token}") {
		return &apperror.Err{
			Code:    "invalid_email_confirmation_path",
			Message: "users.emailConfirmationPath does not contain {token} placeholder",
		}
	}
	confirmationUrl := conf.UString("url") + "/" + strings.Replace(confirmationPath, "{token}", token, -1)

	// Render email.

	subject := conf.UString("users.emailConfirmationSubject", "Confirm your Email")

	var txtContent, htmlContent []byte

	txtTpl := conf.UString("users.emailConfirmationEmailTextTpl")
	htmlTpl := conf.UString("users.emailConfirmationEmailHtmlTpl")
	if txtTpl != "" && htmlTpl != "" {
		// Check that a template engine is configured.
		engine := s.deps.TemplateEngine()
		if engine == nil {
			return apperror.New("no_template_engine")
		}

		data := map[string]interface{}{
			"user":  user,
			"token": token,
		}
		var err apperror.Error

		txtContent, err = s.deps.TemplateEngine().BuildFileAndRender(txtTpl, data)
		if err != nil {
			return apperror.Wrap(err, "email_confirmation_tpl_error", "Could not render email confirmation tpl")
		}

		htmlContent, err = s.deps.TemplateEngine().BuildFileAndRender(htmlTpl, data)
		if err != nil {
			return apperror.Wrap(err, "email_confirmation_tpl_error", "Could not render email confirmation tpl")
		}
	} else {
		tpl := `Welcome to Appkit

To confirm your email address, please visit %v.
`

		htmlTpl := `Welcome to Appkit<br><br>

To confirm your email address, please visit <a href="%v">this link</a>.
`
		txtContent = []byte(fmt.Sprintf(tpl, confirmationUrl))
		htmlContent = []byte(fmt.Sprintf(htmlTpl, confirmationUrl))
	}

	// Now build the email and send it.
	email := email.NewMail()
	email.SetSubject(subject)
	email.AddBody("text/plain", txtContent)
	email.AddBody("text/html", htmlContent)
	email.AddTo(user.GetEmail(), "")

	if err := mailService.Send(email); err != nil {
		return err
	}

	s.deps.Logger().WithFields(logrus.Fields{
		"action":  "users.email_confirmation_mail_sent",
		"email":   user.GetEmail(),
		"user_id": user.GetID(),
		"token":   token,
	}).Debugf("Password reset email sent to %v for user %v", user.GetEmail(), user.GetID())

	return nil
}

func (s *Service) ConfirmEmail(token string) (kit.User, apperror.Error) {
	rawToken, err := s.Tokens.FindOne(token)
	if err != nil {
		return nil, apperror.Wrap(err, "token_query_error", "")
	}
	if rawToken == nil {
		return nil, apperror.New("invalid_token")
	}

	tokenItem := rawToken.(kit.UserToken)
	if !tokenItem.IsValid() {
		return nil, apperror.New("expired_token")
	}

	rawUser, err := s.Users.FindOne(tokenItem.GetUserID())
	if err != nil {
		return nil, apperror.Wrap(err, "user_query_error", "")
	}
	if rawUser == nil {
		return nil, apperror.New("invalid_user")
	}

	user := rawUser.(kit.User)
	userId := user.GetStrID()

	if user.IsEmailConfirmed() {
		// Email already confirmed.
		// Delete tokens and return.
		q := s.Tokens.Q().Filter("user_id", userId).Filter("type", "email_confirmation")
		s.Tokens.Backend().DeleteMany(q)

		return nil, &apperror.Err{
			Code:    "email_already_confirmed",
			Message: "The email is already confirmed",
		}
	}

	user.SetIsEmailConfirmed(true)
	if err := s.Users.Backend().Update(user); err != nil {
		return nil, apperror.Wrap(err, "user_persist_error", "")
	}

	// Delete tokens.
	q := s.Tokens.Q().Filter("user_id", userId).Filter("type", "email_confirmation")
	s.Tokens.Backend().DeleteMany(q)

	s.deps.Logger().WithFields(logrus.Fields{
		"action":  "users.email_confirmed",
		"email":   user.GetEmail(),
		"user_id": user.GetID(),
	}).Debugf("Confirmed email %v for user %v", user.GetEmail(), user.GetID())

	return user, nil
}

func (s *Service) SendPasswordResetEmail(user kit.User) apperror.Error {
	// Check that an email service is configured.

	mailService := s.deps.EmailService()
	if mailService == nil {
		return apperror.New("no_email_service")
	}

	hoursValid := 48

	// Generate a token.
	expiresAt := time.Now().Add(time.Hour * time.Duration(hoursValid))
	tokenItem, err := s.BuildToken("password_reset", user.GetStrID(), expiresAt)
	if err != nil {
		return err
	}
	token := tokenItem.GetToken()

	conf := s.deps.Config()

	// Build the confirmation url.

	resetPath := conf.UString("users.passwordResetPath")
	if resetPath == "" {
		return &apperror.Err{
			Code:    "no_password_reset_path",
			Message: "Config must specify users.passwordResetPath",
		}
	}

	if !strings.Contains(resetPath, "{token}") {
		return &apperror.Err{
			Code:    "invalid_password_reset_path",
			Message: "users.passwordResetPath does not contain {token} placeholder",
		}
	}
	resetUrl := conf.UString("url") + "/" + strings.Replace(resetPath, "{token}", token, -1)

	// Render email.

	subject := conf.UString("users.passwordResetSubject", "Password reset")

	var txtContent, htmlContent []byte

	txtTpl := conf.UString("users.passwordResetTextTpl")
	htmlTpl := conf.UString("users.passwordResetHtmlTpl")
	if txtTpl != "" && htmlTpl != "" {
		// Check that a template engine is configured.
		engine := s.deps.TemplateEngine()
		if engine == nil {
			return apperror.New("no_template_engine")
		}

		data := map[string]interface{}{
			"user":        user,
			"token":       token,
			"hours_valid": hoursValid,
		}
		var err apperror.Error

		txtContent, err = s.deps.TemplateEngine().BuildFileAndRender(txtTpl, data)
		if err != nil {
			return apperror.Wrap(err, "password_reset_tpl_error", "Could not render password reset tpl")
		}

		htmlContent, err = s.deps.TemplateEngine().BuildFileAndRender(htmlTpl, data)
		if err != nil {
			return apperror.Wrap(err, "password_reset_tpl_error", "Could not render password reset tpl")
		}
	} else {
		tpl := `Password reset

To reset your password, please visit %v.
The link will be valid for %v hours.
`

		htmlTpl := `Password Reset<br><br>

To reset your password, please visit <a href="%v">this link</a>.<br>
The link will be valid for %v hours.
`
		txtContent = []byte(fmt.Sprintf(tpl, resetUrl, hoursValid))
		htmlContent = []byte(fmt.Sprintf(htmlTpl, resetUrl, hoursValid))
	}

	// Now build the email and send it.
	email := email.NewMail()
	email.SetSubject(subject)
	email.AddBody("text/plain", txtContent)
	email.AddBody("text/html", htmlContent)
	email.AddTo(user.GetEmail(), "")

	if err := mailService.Send(email); err != nil {
		return err
	}

	s.deps.Logger().WithFields(logrus.Fields{
		"action":  "users.password_reset_requested",
		"email":   user.GetEmail(),
		"user_id": user.GetID(),
		"token":   token,
	}).Debugf("Password reset email sent to %v for user %v", user.GetEmail(), user.GetID())

	return nil
}

func (s *Service) ChangePassword(user kit.User, newPassword string) apperror.Error {
	adaptor := s.AuthAdaptor("password")
	if adaptor == nil {
		return &apperror.Err{
			Code:    "no_password_adaptor",
			Message: "The UserService does not have the password auth adaptor",
		}
	}

	passwordAdaptor := adaptor.(*password.AuthAdaptorPassword)

	if err := passwordAdaptor.ChangePassword(user.GetStrID(), newPassword); err != nil {
		if err.IsPublic() {
			return err
		} else {
			return apperror.Wrap(err, "adapter_error")
		}
		return err
	}

	return nil
}

func (s *Service) ResetPassword(token, newPassword string) (kit.User, apperror.Error) {
	rawToken, err := s.Tokens.FindOne(token)
	if err != nil {
		return nil, apperror.Wrap(err, "token_query_error", "")
	}
	if rawToken == nil {
		return nil, apperror.New("invalid_token")
	}

	tokenItem := rawToken.(kit.UserToken)
	if !tokenItem.IsValid() {
		return nil, apperror.New("expired_token")
	}

	rawUser, err := s.Users.FindOne(tokenItem.GetUserID())
	if err != nil {
		return nil, apperror.Wrap(err, "user_query_error", "")
	}
	if rawUser == nil {
		return nil, apperror.New("invalid_user")
	}
	user := rawUser.(kit.User)

	if err := s.ChangePassword(user, newPassword); err != nil {
		return nil, err
	}

	// Delete token.
	s.Tokens.Backend().Delete(tokenItem)

	s.deps.Logger().WithFields(logrus.Fields{
		"action":  "users.password_reset",
		"user_id": user.GetID(),
	}).Debugf("Password for user %v was reset", user.GetID())

	return user, nil
}

func (h *Service) AuthenticateUser(user kit.User, authAdaptorName string, data map[string]interface{}) (kit.User, apperror.Error) {
	authAdaptor := h.AuthAdaptor(authAdaptorName)
	if authAdaptor == nil {
		return nil, &apperror.Err{
			Public:  true,
			Code:    "unknown_auth_adaptor",
			Message: "Unknown auth adaptor: " + authAdaptorName}
	}

	userId := ""
	if user != nil {
		userId = user.GetStrID()
	}

	var err apperror.Error
	userId, err = authAdaptor.Authenticate(userId, data)
	if err != nil {
		if err.IsPublic() {
			return nil, err
		} else {
			return nil, apperror.Wrap(err, "adaptor_error", true)
		}
	}

	// Query user to get a full user with permissions and profile.

	rawUser, err := h.FindUser(userId)
	if err != nil {
		return nil, apperror.Wrap(err, "user_query_error", "")
	} else if rawUser == nil {
		return nil, &apperror.Err{
			Code:    "user_not_found",
			Message: fmt.Sprintf("User with id %v could not be found", userId),
			Public:  true,
		}
	}
	user = rawUser.(kit.User)

	if !user.IsActive() {
		return nil, apperror.New("user_inactive", true)
	}

	return user, nil
}

func (h *Service) VerifySession(token string) (kit.User, kit.Session, apperror.Error) {
	rawSession, err := h.Sessions.FindOne(token)
	if err != nil {
		return nil, nil, err
	}
	if rawSession == nil {
		return nil, nil, apperror.New("session_not_found")
	}
	session := rawSession.(kit.Session)

	// Load user.
	rawUser, err := h.FindUser(session.GetUserID())
	if err != nil {
		return nil, nil, err
	}
	user := rawUser.(kit.User)

	if !user.IsActive() {
		return nil, nil, apperror.New("user_inactive")
	}

	if session.GetValidUntil().Sub(time.Now()) < 1 {
		return nil, nil, apperror.New("session_expired")
	}

	// Prolong session
	session.SetValidUntil(time.Now().Add(time.Hour * 12))
	if err := h.Sessions.Update(session, nil); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}
