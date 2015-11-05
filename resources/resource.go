package resources

import (
	"math"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"
	"github.com/theduke/go-reflector"

	kit "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/utils"
)

type Resource struct {
	debug    bool
	registry kit.Registry

	backend   db.Backend
	modelInfo *db.ModelInfo
	hooks     interface{}

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

func (res *Resource) Registry() kit.Registry {
	return res.registry
}

func (res *Resource) SetRegistry(x kit.Registry) {
	res.registry = x
}

func (res *Resource) Backend() db.Backend {
	return res.backend
}

func (res *Resource) SetBackend(b db.Backend) {
	res.backend = b

	if !b.HasCollection(res.Collection()) {
		b.RegisterModel(res.Model())
	}
	res.modelInfo = b.ModelInfo(res.Collection())
}

func (res *Resource) ModelInfo() *db.ModelInfo {
	return res.modelInfo
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
	n := res.modelInfo.New()
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
func (res Resource) Query(q *db.Query, targetSlice ...interface{}) ([]kit.Model, apperror.Error) {
	items, err := res.backend.Query(q, targetSlice...)
	if err != nil {
		return nil, err
	}
	return utils.InterfaceToModelSlice(items), nil
}

func (res Resource) Count(q *db.Query) (int, apperror.Error) {
	return res.Backend().Count(q)
}

/**
 * Return a new query initialized with the backend.
 */
func (res Resource) Q() *db.Query {
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
		return kit.NewErrorResponse(err)
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

func (res *Resource) ApiFind(query *db.Query, r kit.Request) kit.Response {
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
		return kit.NewErrorResponse(err)
	}

	user := r.GetUser()
	if allowFind, ok := res.hooks.(AllowFindHook); ok {
		finalItems := make([]kit.Model, 0)
		for _, item := range result {
			if allowFind.AllowFind(res, item, user) {
				finalItems = append(finalItems, item)
			}
		}
		result = finalItems
	}

	response := &kit.AppResponse{
		Data: result,
	}

	// If a limit was set, count the total number of results
	// and set count parameter in metadata.
	limit := query.GetLimit()
	if limit > 0 {
		query.Limit(0).Offset(0)
		count, err := res.backend.Count(query)
		if err != nil {
			return &kit.AppResponse{
				Error: apperror.Wrap(err, "count_error", ""),
			}
		}

		response.SetMeta(map[string]interface{}{
			"count":       count,
			"total_pages": math.Ceil(float64(count) / float64(limit)),
		})
	}

	if hook, ok := res.hooks.(ApiAfterFindHook); ok {
		if err := hook.ApiAfterFind(res, result, r, response); err != nil {
			return kit.NewErrorResponse(err)
		}
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

	// This has to be done before tthe AllowCreate hook to allow the hook to
	// compare UserId value.
	if userModel, ok := obj.(kit.UserModel); ok && user != nil {
		if reflector.R(userModel.GetUserId()).IsZero() {
			userModel.SetUserId(user.GetId())
		}
	}

	if allowCreate, ok := res.hooks.(AllowCreateHook); ok {
		if !allowCreate.AllowCreate(res, obj, user) {
			return apperror.New("permission_denied")
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
		return kit.NewErrorResponse(err)
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

	oldObj, err := res.FindOne(obj.GetId())
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
		rOld := reflector.Reflect(oldObj).MustStruct()
		rNew := reflector.Reflect(oldObj).MustStruct()

		for fieldName, _ := range res.modelInfo.Attributes() {
			val := rNew.Field(fieldName)
			if !val.IsZero() {
				rOld.Field(fieldName).Set(val)
			}
		}
		for fieldName, _ := range res.modelInfo.Relations() {
			val := rNew.Field(fieldName)
			if !val.IsZero() {
				rOld.Field(fieldName).Set(val)
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
		return kit.NewErrorResponse(err)
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
		return kit.NewErrorResponse(err)
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
		return kit.NewErrorResponse(err)
	} else if oldObj == nil {
		return kit.NewErrorResponse("not_found", "")
	}

	user := r.GetUser()
	if err := res.Delete(oldObj, user); err != nil {
		return kit.NewErrorResponse(err)
	}

	return &kit.AppResponse{
		Data: oldObj,
	}
}

// ReadOnlyResource is a resource mixin that prevents all create/update/delete
// actions via the API.
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

// PublicWriteResource is a resource mixin that allows create/update/delete
// to all API users, event without an account.
type PublicWriteResource struct{}

func (PublicWriteResource) AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool {
	return true
}

func (PublicWriteResource) AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool {
	return true
}

func (PublicWriteResource) AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool {
	return true
}

// UserResource is a resource mixin that restricts create, read and update operations to
// admins, users with the permission action_collectionname (see AdminResource) or
// users that own the model.
// This can only be used for models that implement the appkit.UserModel interface.
type UserResource struct{}

func (UserResource) AllowFind(res kit.Resource, model kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if model.(kit.UserModel).GetUserId() == user.GetId() {
		return true
	}
	return user.HasRole("admin")
}

func (UserResource) AllowCreate(res kit.Resource, obj kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if obj.(kit.UserModel).GetUserId() == user.GetId() {
		return true
	}
	return user.HasRole("admin") || user.HasPermission(res.Collection()+".create")
}

func (UserResource) AllowUpdate(res kit.Resource, obj kit.Model, old kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if obj.(kit.UserModel).GetUserId() == user.GetId() {
		return true
	}
	return user.HasRole("admin") || user.HasPermission(res.Collection()+".update")
}

func (UserResource) AllowDelete(res kit.Resource, obj kit.Model, user kit.User) bool {
	if user == nil {
		return false
	}
	if obj.(kit.UserModel).GetUserId() == user.GetId() {
		return true
	}
	return user.HasRole("admin") || user.HasPermission(res.Collection()+".delete")
}
