package appkit

import(
	"strconv"

	db "github.com/theduke/go-dukedb"
)

type BaseUserModelStrID struct {
	db.BaseModelStrID
	
	user ApiUser	
	userID string
}

func(m *BaseUserModelStrID) User() ApiUser {
	return m.user
}

func(m *BaseUserModelStrID) SetUser(x ApiUser) {
	m.user = x
	m.userID = x.GetID()
}

func(m *BaseUserModelStrID) UserID() string {
	return m.userID
}

func(m *BaseUserModelStrID) SetUserID(x string) error {
	m.userID = x
	return nil
}


type BaseUserModelIntID struct {
	db.BaseModelIntID
	
	user ApiUser	
	userID uint64
}

func(m *BaseUserModelIntID) User() ApiUser {
	return m.user
}

func(m *BaseUserModelIntID) SetUser(x ApiUser) {
	m.user = x
	m.SetUserID(x.GetID())
}

func(m *BaseUserModelIntID) UserID() string {
	return strconv.FormatUint(m.userID, 10)
}

func(m *BaseUserModelIntID) SetUserID(rawId string) error {
	id, err := strconv.ParseUint(rawId, 10, 64)	
	if err != nil {
		return err
	}
	m.userID = id
	return nil
}
