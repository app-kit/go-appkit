package users

import (
	"strconv"
	"time"
	"encoding/json"

	kit "github.com/theduke/appkit"
)

type BaseAuthItem struct {
	UserID string `sql:"-" db:"primary_key"`
	Typ string `sql:"size: 100; not null"`

	Data string `sql:type:text; not null`
}

func(a *BaseAuthItem) GetCollection() string {
	return "auth_items"
}

func(a *BaseAuthItem) GetName() string {
	return "auth_items"
}

func (a BaseAuthItem) TableName() string {
	return "auth_items"
}

func(a *BaseAuthItem) GetID() string {
		return ""
}

func(a *BaseAuthItem) SetID(x string) error {
	return nil
}

func (b *BaseAuthItem) SetUserID(rawId string) {
	b.UserID = rawId
}

func (b *BaseAuthItem) GetUserID() string {
	return b.UserID
}

func (b *BaseAuthItem) SetType(x string) {
	b.Typ = x
}

func (b *BaseAuthItem) GetType() string {
	return b.Typ
}

func (b *BaseAuthItem) SetData(data interface{}) error {
	json, err := json.Marshal(data)
	if err != nil {
		return err
	}

	b.Data = string(json)
	return nil
}

func (b *BaseAuthItem) GetData() (interface{}, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(b.Data), &data); err != nil {
		return nil, err
	}

	return data, nil
}

type BaseAuthItemIntID struct {
	ID uint64
	BaseAuthItem
	UserID uint64 `gorm:"primary_key" sql:"not null;"`
}

func (u *BaseAuthItemIntID) SetUserID(x string) {
	i, err := strconv.ParseUint(x, 10, 64)
	if err != nil {
		return
	}

	u.UserID = i
}

func (u *BaseAuthItemIntID) GetUserID() string {
	return strconv.FormatUint(u.UserID, 10)
}


type BaseUser struct {
	Active bool `sql:"not null"`

	Username string `sql:"size:100; not null; unique"`
	Email    string `sql:"size:255; not null; unique"`

	LastLogin time.Time `jsonapi:"name=last-login"`

	CreatedAt time.Time `jsonapi:"name=created-at"`
	UpdatedAt time.Time `jsonapi:"name=updated-at"`

	Roles []*Role `gorm:"many2many:user_roles;" db:"m2m;"`
}

func (u BaseUser) GetCollection() string {
	return "users"
}

// For api2go!
func(a *BaseUser) GetName() string {
	return "users"
}

func (a BaseUser) TableName() string {
	return "users"
}

// Implement User interface.


func (u *BaseUser) SetIsActive(x bool) {
	u.Active = x	
}

func (u *BaseUser) IsActive() bool {
	return u.Active
}

func (u *BaseUser) SetEmail(x string) {
	u.Email = x	
}

func (u *BaseUser) GetEmail() string {
	return u.Email
}

func (u *BaseUser) SetUsername(x string) {
	u.Username = x	
}

func (u *BaseUser) GetUsername() string {
	return u.Username
}

func (u *BaseUser) SetLastLogin(x time.Time) {
	u.LastLogin = x	
}

func (u *BaseUser) GetLastLogin() time.Time {
	return u.LastLogin
}

func (u *BaseUser) SetCreatedAt(x time.Time) {
	u.CreatedAt = x	
}

func (u *BaseUser) GetCreatedAt() time.Time {
	return u.CreatedAt
}

func (u *BaseUser) SetUpdatedAt(x time.Time) {
	u.UpdatedAt = x	
}

func (u *BaseUser) GetUpdatedAt() time.Time {
	return u.UpdatedAt
}

/**
 * RBAC methods.
 */

func (u *BaseUser) GetRoles() []kit.ApiRole {
	slice := make([]kit.ApiRole, 0)
	for _, r := range u.Roles {
		slice = append(slice, r)
	}
	return slice
}

func (u *BaseUser) AddRole(r kit.ApiRole) {
	if u.Roles == nil {
		u.Roles = make([]*Role, 0)
	}
	if !u.HasRole(r) {
		u.Roles = append(u.Roles, r.(*Role))
	}
}

func (u *BaseUser) RemoveRole(r kit.ApiRole) {
	if u.Roles == nil {
		return
	}

	for i := 0; i < len(u.Roles); i++ {
		if u.Roles[i].Name == r.GetName() {
			u.Roles = append(u.Roles[:i], u.Roles[i+1:]...)
		}
	}	
}

func (u *BaseUser) ClearRoles() {
	u.Roles = make([]*Role, 0)
}

func (u *BaseUser) HasRole(r kit.ApiRole) bool {
	return u.HasRoleStr(r.GetName())
}

func (u *BaseUser) HasRoleStr(role string) bool {
	if u.Roles == nil {
		return false
	}

	for i := 0; i < len(u.Roles); i++ {
		if u.Roles[i].Name == role {
			return true
		}
	}

	return false
}


type BaseUserIntID struct {
	BaseUser

	ID uint64 `sql:"not null"`
	Roles []*Role `gorm:"many2many:user_roles;" db:"m2m;"`
}

// For api2go!
func (u BaseUserIntID) GetName() string {
	return "users"
}

func (u *BaseUserIntID) SetID(x string) error {
	i, err := strconv.ParseUint(x, 10, 64)
	if err != nil {
		return err
	}

	u.ID = i
	return nil
}

func (u *BaseUserIntID) GetID() string {
	return strconv.FormatUint(u.ID, 10)
}

/**
 * Base UserProfile.
 */

type BaseUserProfile struct {
	UserID string `sql:"-"`
}

// TODO: fix.
func (p *BaseUserProfile) SetUserID(x string) {
	p.UserID = x
}

func (p BaseUserProfile) GetUserID() string {
	return p.UserID
}

type BaseUserProfileIntID struct {
	UserID uint64 `gorm:"primary_key" sql:"not null"`
}

func (p *BaseUserProfileIntID) SetUserID(x string) {
	id, _ := strconv.ParseUint(x, 10, 64)
	p.UserID = id
}

func (p *BaseUserProfileIntID) GetUserID() string {
	return strconv.FormatUint(p.UserID, 10)
}

/**
 * BaseSession 
 */

type BaseSession struct {
	Token string `gorm:"primary_key" db:"primary_key" sql:"size:100"`
	UserID string `sql:"-"`
	StartedAt  time.Time `sql:"not null" jsonapi:"name=started-at"`
	ValidUntil time.Time `sql:"not null" jsonapi:"name=valid-until"`	

	Typ string `sql:"size:100; not null"`
}

func (b BaseSession) GetCollection() string {
	return "sessions"
}

// For api2go.
func (b BaseSession) GetName() string {
	return "sessions"
}

func (b BaseSession) TableName() string {
	return "sessions"
}

func (s BaseSession) GetID() string {
	return s.Token
}

func (s *BaseSession) SetID(x string) error {
	s.Token = x
	return nil
}

func(s *BaseSession) GetType() string {
	return s.Typ
}

func(s *BaseSession) SetType(x string) {
	s.Typ = x
}

func (s *BaseSession) SetToken(x string) {
	s.Token = x
}

func (s *BaseSession) GetToken() string {
	return s.Token
}

func (s *BaseSession) SetUserID(x string) {
	s.UserID = x
}

func (s *BaseSession) GetUserID() string {
	return s.UserID
}

func (s *BaseSession) SetStartedAt(x time.Time) {
	s.StartedAt = x
}

func (s *BaseSession) GetStartedAt() time.Time {
	return s.StartedAt
}

func (s *BaseSession) SetValidUntil(x time.Time) {
	s.ValidUntil = x
}

func (s *BaseSession) GetValidUntil() time.Time {
	return s.ValidUntil
}

func (s *BaseSession) IsGuest() bool {
	return s.UserID == ""
}

type BaseSessionIntID struct {
	BaseSession
	UserID uint64 `gorm:"primary_key" sql:"not null;"`
}

func (u *BaseSessionIntID) SetUserID(x string) {
	i, _ := strconv.ParseUint(x, 10, 64)
	u.UserID = i
}

func (u *BaseSessionIntID) GetUserID() string {
	return strconv.FormatUint(u.UserID, 10)
}

func (s *BaseSessionIntID) IsGuest() bool {
	return s.UserID == 0
}

/**
 * Role.
 */

type Role struct {
	Name string `gorm:"primary_key" db:"primary_key" sql:"type: varchar(200)"`
	Permissions []*Permission `gorm:"many2many:role_permissions;" db:"m2m"`
}

func (r Role) GetCollection() string {
	return "roles"
}

func (r Role) GetTableName() string {
	return "roles"
}

func (r *Role) SetName(n string) {
	r.Name = n
}

func (r Role) GetName() string {
	return r.Name
}

func (r Role) GetID() string {
	return r.Name
}

func (r *Role) SetID(n string) error {
	r.Name = n
	return nil
} 

/**
 * Permission.
 */

type Permission struct {
	Name string `gorm:"primary_key" db:"primary_key" sql:"type: varchar(200)"`
}

func (r Permission) GetCollection() string {
	return "permissions"
}

func (r Permission) GetTableName() string {
	return "permissions"
}

func (r *Permission) SetName(n string) {
	r.Name = n
}

func (r Permission) GetName() string {
	return r.Name
}

func (p Permission) GetID() string {
	return p.Name
}

func (p *Permission) SetID(n string) error {
	p.Name = n
	return nil
} 