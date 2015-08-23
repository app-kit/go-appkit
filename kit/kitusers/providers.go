package kitusers 

import(
	//"log"
	"strconv"
	"reflect"
	
	"github.com/jinzhu/gorm"

	"github.com/theduke/appkit/users"
)

func asInt(x string) uint64 {
	id, _ := strconv.ParseUint(x, 10, 64)
	return id
}

func NewUserHandler(db *gorm.DB, profileType interface{}) *users.BaseUserHandler {
	handler := users.NewUserHandler()
	handler.SetSessionProvider(SessionProvider{Db: db})
	handler.SetUserProvider(UserProvider{Db: db, ProfileType: profileType})
	handler.SetAuthProvider(AuthProvider{Db: db})

	return handler
}

/**
 * AuthProvider.
 */

type AuthProvider struct {
	Db *gorm.DB
}

func (p AuthProvider) NewItem(userId string, typ string, data interface{}) users.AuthItem {
	item := AuthItem{}
	item.SetUserID(userId)
	item.SetType(typ)
	item.SetData(data)

	return &item
}

func (p AuthProvider) Create(item users.AuthItem) error {
	return p.Db.Create(item).Error
}

func (p AuthProvider) Update(item users.AuthItem) error {
	return p.Db.Save(item).Error
}

func (p AuthProvider) Delete(item users.AuthItem) error {
	return p.Db.Delete(item).Error
}

func (p AuthProvider) FindOne(typ string, user users.User) (users.AuthItem, error) {
	var item AuthItem
	if err := p.Db.Where("typ = ? AND user_id = ?", typ, asInt(user.GetID())).First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (p AuthProvider) FindByUser(user users.User) ([]users.AuthItem, error) {
	var items []AuthItem
	if err := p.Db.Where("user_id = ?", asInt(user.GetID())).Error; err != nil {
		return nil, err
	}


	list := make([]users.AuthItem, 0)
	for _, item := range items {
		list = append(list, &item)
	}

	return list, nil
}


/**
 * UserProvider.
 */

type UserProvider struct {
	Db *gorm.DB
	ProfileType interface{}
}

func (p UserProvider) NewUser() users.User {
	user := User{}
	return &user
}

func (p UserProvider) Create(item users.User) error {
	if err := p.Db.Create(item).Error; err != nil {
		return err
	}

	if p.ProfileType != nil {
		var profile interface{}

		profileField := reflect.ValueOf(item).Elem().FieldByName("Profile")
		if profileField.Elem().IsValid() {
			profile = profileField.Elem().Addr().Interface()
		} else {
			profile = reflect.ValueOf(p.ProfileType).Elem().Addr().Interface()
		}

		reflect.ValueOf(profile).Elem().FieldByName("UserID").SetUint(asInt(item.GetID()))

		if err := p.Db.Create(profile).Error; err != nil {
			// Saving profile failed, delete user.
			errDelete := p.Db.Delete(item).Error
			if errDelete != nil {
				return errDelete
			}

			return err
		}
	}

	return nil
}

func (p UserProvider) Update(item users.User) error {
	err :=  p.Db.Save(item).Error
	if err != nil {
		return err
	}

	if p.ProfileType != nil {
		profile := reflect.ValueOf(item).Elem().FieldByName("Profile").Elem().Addr().Interface()
		return p.Db.Save(profile).Error
	}

	return err
}

func (p UserProvider) Delete(item users.User) error {
	if p.ProfileType != nil {
		profile := reflect.ValueOf(item).Elem().FieldByName("Profile").Elem().Addr().Interface()
		return p.Db.Delete(profile).Error
	}

	return p.Db.Delete(item).Error
}

func (p UserProvider) FindOne(id string) (users.User, error) {
	var item User
	if err := p.Db.Where("id = ?", asInt(id)).First(&item).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return &item, nil
}

func (p UserProvider) FindByEmail(email string) (users.User, error) {
	var item User
	if err := p.Db.Where("email = ?", email).First(&item).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return &item, nil
}

func (p UserProvider) FindByUsername(username string) (users.User, error) {
	var item User
	if err := p.Db.Where("username = ?", username).First(&item).Error; err != nil {
		if err.Error() == "record not found" {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return &item, nil
}

func (p UserProvider) FindAll() ([]users.User, error) {
	var items []User
	if err := p.Db.Find(&items).Error; err != nil {
		return nil, err
	}

	list := make([]users.User, 0)
	for _, item := range items {
		list = append(list, &item)
	}

	return list, nil
}

/**
 * Session.
 */

type SessionProvider struct {
	Db *gorm.DB
}

func (p SessionProvider) NewSession() users.Session {
	item := Session{}
	return &item
}

func (p SessionProvider) Create(item users.Session) error {
	return p.Db.Create(item).Error
}

func (p SessionProvider) Update(item users.Session) error {
	return p.Db.Save(item).Error
}

func (p SessionProvider) Delete(item users.Session) error {
	return p.Db.Delete(item).Error
}

func (p SessionProvider) FindOne(id string) (users.Session, error) {
	var item Session
	if err := p.Db.Where("id = ?", asInt(id)).First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (p SessionProvider) FindAll() ([]users.Session, error) {
	var items []Session
	if err := p.Db.Find(&items).Error; err != nil {
		return nil, err
	}

	list := make([]users.Session, 0)
	for _, item := range items {
		list = append(list, &item)
	}

	return list, nil
}
