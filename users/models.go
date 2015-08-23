package users

import (
	"strconv"
	"time"
	"encoding/json"
)

type BaseAuthItem struct {
	UserID string `sql:"-"`
	Typ string `sql:"size: 100; not null"`

	Data string `sql:type:text; not null`
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
	BaseAuthItem
	UserID uint64
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
	ID string `sql:"-"`

	Active bool `sql:"not null"`

	Username string `sql:"size:100; not null; unique"`
	Email    string `sql:"size:255; not null; unique"`

	LastLogin time.Time `jsonapi:"name=last-login"`

	CreatedAt time.Time `jsonapi:"name=created-at"`
	UpdatedAt time.Time `jsonapi:"name=updated-at"`
}

/**
 * Implements api2go interface.
 * See https://github.com/manyminds/api2go
 */
func (u BaseUser) GetName() string {
	return "users"
}

// Implement User interface.

func (u *BaseUser) SetID(x string) error {
	u.ID = x
	return nil
}

func (u *BaseUser) GetID() string {
	return u.ID
}

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


type BaseUserIntID struct {
	BaseUser

	ID uint64 `gorm:"primary_key"`
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
	Token string `gorm:"primary_key" sql:"size:100"`
	UserID string `sql:"-"`
	StartedAt  time.Time `sql:"not null" jsonapi:"name=started-at"`
	ValidUntil time.Time `sql:"not null" jsonapi:"name=valid-until"`	
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
	UserID uint64 `sql:"not null;"`
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