package users

import (
	"fmt"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	db "github.com/theduke/go-dukedb"
	"github.com/twinj/uuid"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/resources"
	"github.com/theduke/go-appkit/users/auth"
)

type Service struct {
	debug bool
	deps  kit.Dependencies

	backend db.Backend

	Users     kit.Resource
	Sessions  kit.Resource
	AuthItems kit.Resource
	Tokens    kit.Resource

	Roles       kit.Resource
	Permissions kit.Resource

	profileModel kit.UserProfile

	AuthAdaptors map[string]kit.AuthAdaptor
}

// Ensure UserService implements kit.UserService.
var _ kit.UserService = (*Service)(nil)

func NewService(deps kit.Dependencies, profileModel kit.UserProfile) *Service {
	h := Service{
		deps:         deps,
		profileModel: profileModel,
	}

	h.AuthAdaptors = make(map[string]kit.AuthAdaptor)

	// Register auth adaptors.
	h.AddAuthAdaptor(auth.AuthAdaptorPassword{})

	// Build resources.
	users := resources.NewResource(&BaseUserIntID{}, UserResourceHooks{
		ProfileModel: profileModel,
	})
	h.Users = users

	h.Tokens = resources.NewResource(&Token{}, nil)

	sessions := resources.NewResource(&BaseSessionIntID{}, SessionResourceHooks{})
	h.Sessions = sessions

	auths := resources.NewResource(&BaseAuthItemIntID{}, nil)
	h.AuthItems = auths

	roles := resources.NewResource(&Role{}, RoleResourceHooks{})
	h.Roles = roles

	permissions := resources.NewResource(&Permission{}, PermissionResourceHooks{})
	h.Permissions = permissions

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

	s.Sessions.SetBackend(b)
	b.RegisterModel(s.Sessions.Model())

	s.AuthItems.SetBackend(b)
	b.RegisterModel(s.AuthItems.Model())

	s.Tokens.SetBackend(b)
	b.RegisterModel(s.Tokens.Model())

	s.Roles.SetBackend(b)
	b.RegisterModel(s.Roles.Model())

	s.Permissions.SetBackend(b)
	b.RegisterModel(s.Permissions.Model())

	if s.profileModel != nil {
		b.RegisterModel(s.profileModel)
	}
	s.backend = b
}

func (h *Service) AuthAdaptor(name string) kit.AuthAdaptor {
	return h.AuthAdaptors[name]
}

func (h *Service) AddAuthAdaptor(a kit.AuthAdaptor) {
	h.AuthAdaptors[a.GetName()] = a
}

func (h *Service) UserResource() kit.Resource {
	return h.Users
}

func (h *Service) SetUserResource(x kit.Resource) {
	h.Users = x
}

func (h *Service) SessionResource() kit.Resource {
	return h.Sessions
}

func (h *Service) SetSessionResource(x kit.Resource) {
	h.Sessions = x
}

func (h *Service) AuthItemResource() kit.Resource {
	return h.AuthItems
}

func (h *Service) SetAuthItemResource(x kit.Resource) {
	h.AuthItems = x
}

func (h *Service) TokenResource() kit.Resource {
	return h.Tokens
}

func (h *Service) SetTokenResource(x kit.Resource) {
	h.Tokens = x
}

func (h *Service) ProfileModel() kit.UserProfile {
	return h.profileModel
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

func (s *Service) BuildToken(typ, userId string, expiresAt time.Time) (kit.UserToken, kit.Error) {
	token := uuid.NewV4().String()

	tokenItem := s.Tokens.NewModel().(kit.UserToken)
	tokenItem.SetType(typ)
	tokenItem.SetToken(token)
	tokenItem.SetUserID(userId)

	if err := s.Tokens.Create(tokenItem, nil); err != nil {
		return nil, kit.WrapError(err, "token_create_error", "Could not save token to database")
	}

	return tokenItem, nil
}

func (s *Service) CreateUser(user kit.User, adaptorName string, authData interface{}) kit.Error {
	adaptor := s.AuthAdaptor(adaptorName)
	if adaptor == nil {
		return kit.AppError{Code: "unknown_auth_adaptor"}
	}

	data, err := adaptor.BuildData(user, authData)
	if err != nil {
		return kit.AppError{Code: "adaptor_error", Message: err.Error()}
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
		return kit.AppError{
			Code:    "user_exists",
			Message: "A user with the username or email already exists",
		}
	}

	user.SetIsActive(true)

	if s.profileModel != nil && user.GetProfile() == nil {
		newProfile, _ := s.Users.Backend().NewModel(s.profileModel.Collection())
		user.SetProfile(newProfile.(kit.UserProfile))
	}

	if err := s.Users.Create(user, nil); err != nil {
		return err
	}

	// Create profile if one exists.
	if profile := user.GetProfile(); profile != nil {
		profile.SetID(user.GetID())
		if err := s.Users.Backend().Create(profile); err != nil {
			s.Users.Backend().Delete(user)
			return err
		}
	}

	rawAuth := s.AuthItems.NewModel()
	auth := rawAuth.(kit.AuthItem)
	auth.SetUserID(user.GetID())
	auth.SetType(adaptorName)
	auth.SetData(data)

	if err := s.AuthItems.Create(auth, nil); err != nil {
		s.Users.Delete(user, nil)
		return kit.AppError{Code: "auth_save_failed", Message: err.Error()}
	}

	if err := s.SendConfirmationEmail(user); err != nil {
		s.deps.Logger().Errorf("Could not send confirmation email: %v", err)
	}

	return nil
}

func (s *Service) SendConfirmationEmail(user kit.User) kit.Error {
	// Check that an email service is configured.

	mailService := s.deps.EmailService()
	if mailService == nil {
		return kit.AppError{Code: "no_email_service"}
	}

	conf := s.deps.Config()

	// Check that sending is enabled.
	if !conf.UBool("users.sendEmailConfirmationEmail", true) {
		return nil
	}

	// Generate a token.
	tokenItem, err := s.BuildToken("email_confirmation", user.GetID(), time.Time{})
	if err != nil {
		return err
	}
	token := tokenItem.GetToken()

	// Build the confirmation url.

	confirmationPath := conf.UString("users.emailConfirmationPath")
	if confirmationPath == "" {
		return kit.AppError{
			Code:     "no_email_confirmation_path",
			Message:  "Config must specify users.emailConfirmationPath",
			Internal: true,
		}
	}

	if !strings.Contains(confirmationPath, "{token}") {
		return kit.AppError{
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
			return kit.AppError{Code: "no_template_engine"}
		}

		data := map[string]interface{}{
			"user":  user,
			"token": token,
		}
		var err kit.Error

		txtContent, err = s.deps.TemplateEngine().BuildFileAndRender(txtTpl, data)
		if err != nil {
			return kit.WrapError(err, "email_confirmation_tpl_error", "Could not render email confirmation tpl")
		}

		htmlContent, err = s.deps.TemplateEngine().BuildFileAndRender(htmlTpl, data)
		if err != nil {
			return kit.WrapError(err, "email_confirmation_tpl_error", "Could not render email confirmation tpl")
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

func (s *Service) ConfirmEmail(token string) (kit.User, kit.Error) {
	rawToken, err := s.Tokens.FindOne(token)
	if err != nil {
		return nil, kit.WrapError(err, "token_query_error", "")
	}
	if rawToken == nil {
		return nil, kit.AppError{Code: "invalid_token"}
	}

	tokenItem := rawToken.(kit.UserToken)
	if !tokenItem.IsValid() {
		return nil, kit.AppError{Code: "expired_token"}
	}

	rawUser, err := s.Users.FindOne(tokenItem.GetUserID())
	if err != nil {
		return nil, kit.WrapError(err, "user_query_error", "")
	}
	if rawUser == nil {
		return nil, kit.AppError{Code: "invalid_user"}
	}

	user := rawUser.(kit.User)
	userId, _ := db.GetStructFieldValue(user, "ID")

	if user.IsEmailConfirmed() {
		// Email already confirmed.
		// Delete tokens and return.
		q := s.Tokens.Q().Filter("user_id", userId).Filter("type", "email_confirmation")
		s.Tokens.Backend().DeleteMany(q)

		return nil, kit.AppError{
			Code:    "email_already_confirmed",
			Message: "The email is already confirmed",
		}
	}

	user.SetIsEmailConfirmed(true)
	if err := s.Users.Backend().Update(user); err != nil {
		return nil, kit.WrapError(err, "user_persist_error", "")
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

func (s *Service) SendPasswordResetEmail(user kit.User) kit.Error {
	// Check that an email service is configured.

	mailService := s.deps.EmailService()
	if mailService == nil {
		return kit.AppError{Code: "no_email_service"}
	}

	hoursValid := 48

	// Generate a token.
	expiresAt := time.Now().Add(time.Hour * time.Duration(hoursValid))
	tokenItem, err := s.BuildToken("password_reset", user.GetID(), expiresAt)
	if err != nil {
		return err
	}
	token := tokenItem.GetToken()

	conf := s.deps.Config()

	// Build the confirmation url.

	resetPath := conf.UString("users.passwordResetPath")
	if resetPath == "" {
		return kit.AppError{
			Code:     "no_password_reset_path",
			Message:  "Config must specify users.passwordResetPath",
			Internal: true,
		}
	}

	if !strings.Contains(resetPath, "{token}") {
		return kit.AppError{
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
			return kit.AppError{Code: "no_template_engine"}
		}

		data := map[string]interface{}{
			"user":        user,
			"token":       token,
			"hours_valid": hoursValid,
		}
		var err kit.Error

		txtContent, err = s.deps.TemplateEngine().BuildFileAndRender(txtTpl, data)
		if err != nil {
			return kit.WrapError(err, "password_reset_tpl_error", "Could not render password reset tpl")
		}

		htmlContent, err = s.deps.TemplateEngine().BuildFileAndRender(htmlTpl, data)
		if err != nil {
			return kit.WrapError(err, "password_reset_tpl_error", "Could not render password reset tpl")
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

func (s *Service) ChangePassword(user kit.User, newPassword string) kit.Error {
	rawAuth, err := s.AuthItems.Q().
		Filter("typ", "password").And("user_id", user.GetID()).First()

	if err != nil {
		return kit.WrapError(err, "auth_item_query_error", "")
	}
	if rawAuth == nil {
		return kit.AppError{Code: "no_password_auth", Message: "User is not configured for password authentication"}
	}

	auth := rawAuth.(kit.AuthItem)

	adaptor := s.AuthAdaptor("password")
	data, err := adaptor.BuildData(user, map[string]interface{}{"password": newPassword})
	if err != nil {
		return err
	}

	auth.SetData(data)
	if err := s.AuthItems.Backend().Update(auth); err != nil {
		return err
	}

	return nil
}

func (s *Service) ResetPassword(token, newPassword string) (kit.User, kit.Error) {
	rawToken, err := s.Tokens.FindOne(token)
	if err != nil {
		return nil, kit.WrapError(err, "token_query_error", "")
	}
	if rawToken == nil {
		return nil, kit.AppError{Code: "invalid_token"}
	}

	tokenItem := rawToken.(kit.UserToken)
	if !tokenItem.IsValid() {
		return nil, kit.AppError{Code: "expired_token"}
	}

	rawUser, err := s.Users.FindOne(tokenItem.GetUserID())
	if err != nil {
		return nil, kit.WrapError(err, "user_query_error", "")
	}
	if rawUser == nil {
		return nil, kit.AppError{Code: "invalid_user"}
	}
	user := rawUser.(kit.User)

	if err := s.ChangePassword(user, newPassword); err != nil {
		return nil, err
	}

	s.deps.Logger().WithFields(logrus.Fields{
		"action":  "users.password_reset",
		"user_id": user.GetID(),
	}).Debugf("Password for user %v was reset", user.GetID())

	return user, nil
}

func (h *Service) AuthenticateUser(user kit.User, authAdaptorName string, data interface{}) kit.Error {
	if !user.IsActive() {
		return kit.AppError{Code: "user_inactive"}
	}

	authAdaptor := h.AuthAdaptor(authAdaptorName)
	if authAdaptor == nil {
		return kit.AppError{
			Code:    "unknown_auth_adaptor",
			Message: "Unknown auth adaptor: " + authAdaptorName}
	}

	rawAuth, err := h.AuthItems.Q().
		Filter("typ", authAdaptorName).And("user_id", user.GetID()).First()

	if err != nil {
		return err
	}

	auth := rawAuth.(kit.AuthItem)

	cleanData, err2 := auth.GetData()
	if err2 != nil {
		return kit.AppError{
			Code:    "invalid_auth_data",
			Message: err.Error(),
		}
	}

	ok, err2 := authAdaptor.Authenticate(user, cleanData, data)
	if err2 != nil {
		return kit.AppError{Code: "auth_error", Message: err.Error()}
	}
	if !ok {
		return kit.AppError{Code: "invalid_credentials"}
	}

	return nil
}

func (h *Service) VerifySession(token string) (kit.User, kit.Session, kit.Error) {
	rawSession, err := h.Sessions.FindOne(token)
	if err != nil {
		return nil, nil, err
	}
	if rawSession == nil {
		return nil, nil, kit.AppError{Code: "session_not_found"}
	}
	session := rawSession.(kit.Session)

	// Load user.
	rawUser, err := h.UserResource().FindOne(session.GetUserID())
	if err != nil {
		return nil, nil, err
	}
	user := rawUser.(kit.User)

	if !user.IsActive() {
		return nil, nil, kit.AppError{Code: "user_inactive"}
	}

	if session.GetValidUntil().Sub(time.Now()) < 1 {
		return nil, nil, kit.AppError{Code: "session_expired"}
	}

	// Prolong session
	session.SetValidUntil(time.Now().Add(time.Hour * 12))
	if err := h.Sessions.Update(session, nil); err != nil {
		return nil, nil, err
	}

	return user, session, nil
}
