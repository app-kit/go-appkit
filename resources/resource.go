package resources

import (
	"reflect"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/utils"
)

type Resource struct {
	debug bool
	deps  kit.Dependencies

	backend db.Backend
	hooks   interface{}

	isPublic bool

	model kit.Model
}

// Ensure Resource implements Resource interface.
var _ kit.Resource = (*Resource)(nil)

func NewResource(model kit.Model, hooks interface{}, isPublic bool) *Resource {
	r := Resource{
		isPublic: isPublic,
	}

	r.SetModel(model)

	if hooks == nil {
		hooks = &LoggedInResource{}
	}

	r.SetHooks(hooks)
	return &r
}

func (res *Resource) Debug() bool {
	return res.debug
}

func (res *Resource) SetDebug(x bool) {
	res.debug = x
}

func (res *Resource) Dependencies() kit.Dependencies {
	return res.deps
}

func (res *Resource) SetDependencies(x kit.Dependencies) {
	res.deps = x
}

func (res *Resource) Backend() db.Backend {
	return res.backend
}

func (res *Resource) SetBackend(b db.Backend) {
	res.backend = b

	if !b.HasCollection(res.Collection()) {
		b.RegisterModel(res.Model())
	}
}

func (res *Resource) IsPublic() bool {
	return res.isPublic
}

func (res *Resource) Collection() string {
	return res.model.Collection()
}

func (res *Resource) Model() kit.Model {
	return res.model
}

func (res *Resource) SetModel(x kit.Model) {
	res.model = x
}

func (res *Resource) CreateModel() kit.Model {
	n, err := res.backend.CreateModel(res.model.Collection())
	if err != nil {

	}
	return n.(kit.Model)
}

func (res *Resource) Hooks() interface{} {
	return res.hooks
}

func (res *Resource) SetHooks(h interface{}) {
	res.hooks = h
}

/**
 * Queries.
 */

/**
 * Perform a query.
 */
func (res Resource) Query(q db.Query, targetSlice ...interface{}) ([]kit.Model, apperror.Error) {
	items, err := res.backend.Query(q, targetSlice...)
	if err != nil {
		return nil, err
	}
	return utils.InterfaceToModelSlice(items), nil
}

func (res Resource) Count(q db.Query) (int, apperror.Error) {
	return res.Backend().Count(q)
}

/**
 * Return a new query initialized with the backend.
 */
func (res Resource) Q() db.Query {
	return res.backend.Q(res.model.Collection())
}

/**
 * FindOne
 */

func (res *Resource) FindOne(rawId interface{}) (kit.Model, apperror.Error) {
	item, err := res.backend.FindOne(res.model.Collection(), rawId)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return item.(kit.Model), nil
}

/**
 * Find.
 */

func (res *Resource) ApiFindOne(rawId string, r kit.Request) kit.Response {
	hook, ok := res.hooks.(ApiFindOneHook)
	if ok {
		return hook.ApiFindOne(res, rawId, r)
	}

	result, err := res.FindOne(rawId)
	if err != nil {
		return &kit.AppResponse{Error: err}
	} else if result == nil {
		return kit.NewErrorResponse("not_found", "")
	}

	user := r.GetUser()
	if allowFind, ok := res.hooks.(AllowFindHook); ok {
		if !allowFind.AllowFind(res, result, user) {
			return kit.NewErrorResponse("permission_denied", "")
		}
	}

	return &kit.AppResponse{
		Data: result,
	}
}

func (res *Resource) ApiFind(query db.Query, r kit.Request) kit.Response {
	// If query is empty, query for all records.
	if query == nil {
		query = res.Q()
	}

	apiFindHook, ok := res.hooks.(ApiFindHook)
	if ok {
		return apiFindHook.ApiFind(res, query, r)
	}

	if alterQuery, ok := res.hooks.(ApiAlterQueryHook); ok {
		alterQuery.ApiAlterQuery(res, query, r)
	}

	result, err := res.Query(query)
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	user := r.GetUser()
	if allowFind, ok := res.hooks.(AllowFindHook); ok {
		for _, item := range result {
			if !allowFind.AllowFind(res, item, user) {
				return kit.NewErrorResponse("permission_denied", "")
			}
		}
	}

	response := &kit.AppResponse{
		Data: result,
	}

	// If a limit was set, count the total number of results
	// and set count parameter in metadata.
	if query.GetLimit() > 0 {
		query.Limit(0).Offset(0)
		count, err := res.backend.Count(query)
		if err != nil {
			return &kit.AppResponse{
				Error: apperror.Wrap(err, "count_error", ""),
			}
		}

		response.SetMeta(map[string]interface{}{"count": count})
	}

	return response
}

/**
 * Create.
 */

func (res *Resource) Create(obj kit.Model, user kit.User) apperror.Error {
	if hook, ok := res.hooks.(CreateHook); ok {
		return hook.Create(res, obj, user)
	}

	if allowCreate, ok := res.hooks.(AllowCreateHook); ok {
		if !allowCreate.AllowCreate(res, obj, user) {
			return apperror.New("permission_denied")
		}
	}

	if userModel, ok := obj.(kit.UserModel); ok && user != nil {
		if db.IsZero(userModel.GetUserID()) {
			userModel.SetUserID(user.GetID())
		}
	}

	if beforeCreate, ok := res.hooks.(BeforeCreateHook); ok {
		if err := beforeCreate.BeforeCreate(res, obj, user); err != nil {
			return err
		}
	}

	if err := res.backend.Create(obj); err != nil {
		return err
	}

	if afterCreate, ok := res.hooks.(AfterCreateHook); ok {
		if err := afterCreate.AfterCreate(res, obj, user); err != nil {
			return err
		}
	}

	return nil
}

func (res *Resource) ApiCreate(obj kit.Model, r kit.Request) kit.Response {
	if createHook, ok := res.hooks.(ApiCreateHook); ok {
		return createHook.ApiCreate(res, obj, r)
	}

	user := r.GetUser()
	err := res.Create(obj, user)
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	return &kit.AppResponse{
		Data: obj,
	}
}

/**
 * Update.
 */

func (res *Resource) update(obj kit.Model, user kit.User, partial bool) apperror.Error {
	if hook, ok := res.hooks.(UpdateHook); ok {
		return hook.Update(res, obj, user)
	}

	oldObj, err := res.FindOne(obj.GetID())
	if err != nil {
		return err
	} else if oldObj == nil {
		return apperror.New("not_found")
	}

	if allowUpdate, ok := res.hooks.(AllowUpdateHook); ok {
		if !allowUpdate.AllowUpdate(res, obj, oldObj, user) {
			return apperror.New("permission_denied")
		}
	}

	if beforeUpdate, ok := res.hooks.(BeforeUpdateHook); ok {
		if err := beforeUpdate.BeforeUpdate(res, obj, oldObj, user); err != nil {
			return err
		}
	}

	if partial {
		info := res.backend.ModelInfo(obj.Collection())

		reflNewObj := reflect.ValueOf(obj).Elem()
		reflOldObj := reflect.ValueOf(oldObj).Elem()

		for fieldName := range info.FieldInfo {
			reflVal := reflNewObj.FieldByName(fieldName)

			if !db.IsZero(reflVal.Interface()) {
				reflOldObj.FieldByName(fieldName).Set(reflVal)
			}
		}

		obj = oldObj
	}

	if err := res.backend.Update(obj); err != nil {
		return err
	}

	if afterUpdate, ok := res.hooks.(AfterUpdateHook); ok {
		if err := afterUpdate.AfterUpdate(res, obj, oldObj, user); err != nil {
			return err
		}
	}

	return nil
}

func (res *Resource) Update(obj kit.Model, user kit.User) apperror.Error {
	return res.update(obj, user, false)

}

func (res *Resource) PartialUpdate(obj kit.Model, user kit.User) apperror.Error {
	return res.update(obj, user, true)
}

func (res *Resource) ApiUpdate(obj kit.Model, r kit.Request) kit.Response {
	if updateHook, ok := res.hooks.(ApiUpdateHook); ok {
		return updateHook.ApiUpdate(res, obj, r)
	}

	user := r.GetUser()
	err := res.Update(obj, user)
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	return &kit.AppResponse{
		Data: obj,
	}
}

func (res *Resource) ApiPartialUpdate(obj kit.Model, r kit.Request) kit.Response {
	if updateHook, ok := res.hooks.(ApiUpdateHook); ok {
		return updateHook.ApiUpdate(res, obj, r)
	}

	user := r.GetUser()
	err := res.PartialUpdate(obj, user)
	if err != nil {
		return &kit.AppResponse{Error: err}
	}

	return &kit.AppResponse{
		Data: obj,
	}
}

/**
 * Delete.
 */

func (res *Resource) Delete(obj kit.Model, user kit.User) apperror.Error {
	if hook, ok := res.hooks.(DeleteHook); ok {
		return hook.Delete(res, obj, user)
	}

	if allowDelete, ok := res.hooks.(AllowDeleteHook); ok {
		if !allowDelete.AllowDelete(res, obj, user) {
			return apperror.New("permission_denied")
		}
	}

	if beforeDelete, ok := res.hooks.(BeforeDeleteHook); ok {
		if err := beforeDelete.BeforeDelete(res, obj, user); err != nil {
			return err
		}
	}

	if err := res.backend.Delete(obj); err != nil {
		return err
	}

	if afterDelete, ok := res.hooks.(AfterDeleteHook); ok {
		if err := afterDelete.AfterDelete(res, obj, user); err != nil {
			return err
		}
	}

	return nil
}

func (res *Resource) ApiDelete(id string, r kit.Request) kit.Response {
	if deleteHook, ok := res.hooks.(ApiDeleteHook); ok {
		return deleteHook.ApiDelete(res, id, r)
	}

	oldObj, err := res.FindOne(id)
	if err != nil {
		return &kit.AppResponse{Error: err}
	} else if oldObj == nil {
		return kit.NewErrorResponse("not_found", "")
	}

	user := r.GetUser()
	if err := res.Delete(oldObj, user); err != nil {
		return &kit.AppResponse{Error: err}
	}

	return &kit.AppResponse{
		Data: oldObj,
	}
}

/**
 * Read only resource hooks template
 */

type ReadOnlyResource struct{}

func (r ReadOnlyResource) AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool {
	return false
}

func (r ReadOnlyResource) AllowUpdate(res kit.Resource, obj kit.Model, user kit.User) bool {
	return false
}

func (r ReadOnlyResource) AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool {
	return false
}

// AdminResource is a mixin that restricts create, update and delete.
// Only users with the role admin or with the permission action_collectionname
// may create, update or delete objects.
// So if you want to allow a role to update all items in a collection
// "totos", the permission update_todos to the role.
type AdminResource struct{}

func (AdminResource) AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool {
	return user != nil && (user.HasRole("admin") || user.HasPermission(res.Collection()+".create"))
}

func (AdminResource) AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool {
	return user != nil && (user.HasRole("admin") || user.HasPermission(res.Collection()+".update"))
}

func (AdminResource) AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool {
	return user != nil && (user.HasRole("admin") || user.HasPermission(res.Collection()+".delete"))
}

// UserResource is a resource mixin that restricts create, read and update operations to
// admins, users with the permission action_collectionname (see AdminResource) or
// users that own the model.
// This can only be used for models that implement the appkit.UserModel interface.
type UserResource struct{}

func (UserResource) AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if obj.(kit.UserModel).GetUserID() == user.GetID() {
		return true
	}
	return user.HasRole("admin") || user.HasPermission(res.Collection()+".create")
}

func (UserResource) AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if obj.(kit.UserModel).GetUserID() == user.GetID() {
		return true
	}
	return user.HasRole("admin") || user.HasPermission(res.Collection()+".update")
}

func (UserResource) AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if obj.(kit.UserModel).GetUserID() == user.GetID() {
		return true
	}
	return user.HasRole("admin") || user.HasPermission(res.Collection()+".delete")
}

// LoggedInResource is a resource mixin that restricts create, read and update operations to
// logged in users.
type LoggedInResource struct{}

func (LoggedInResource) AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool {
	return user != nil
}

func (LoggedInResource) AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool {
	return user != nil
}

func (LoggedInResource) AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool {
	return user != nil
}
