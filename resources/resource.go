package resources

import (
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type Resource struct {
	debug bool
	deps  kit.Dependencies

	backend db.Backend
	hooks   interface{}

	isPublic bool

	model db.Model
}

// Ensure Resource implements Resource interface.
var _ kit.Resource = (*Resource)(nil)

func NewResource(model db.Model, hooks interface{}, isPublic bool) *Resource {
	r := Resource{
		isPublic: isPublic,
	}

	r.SetModel(model)
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

func (res *Resource) SetBackend(x db.Backend) {
	res.backend = x
}

func (res *Resource) IsPublic() bool {
	return res.isPublic
}

func (res *Resource) Collection() string {
	return res.model.Collection()
}

func (res *Resource) Model() db.Model {
	return res.model
}

func (res *Resource) SetModel(x db.Model) {
	res.model = x
}

func (res *Resource) NewModel() db.Model {
	n, err := res.backend.NewModel(res.model.Collection())
	if err != nil {
		return nil
	}
	return n.(db.Model)
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
func (res Resource) Query(q db.Query) ([]db.Model, kit.Error) {
	return res.backend.Query(q)
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

func (res *Resource) FindOne(rawId string) (db.Model, kit.Error) {
	return res.backend.FindOne(res.model.Collection(), rawId)
}

/**
 * Find.
 */

func (res Resource) Find(query db.Query) ([]db.Model, kit.Error) {
	return res.backend.Query(query)
}

func (res *Resource) ApiFindOne(rawId string, r kit.Request) kit.Response {
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
				Error: kit.WrapError(err, "count_error", ""),
			}
		}

		response.SetMeta(map[string]interface{}{"count": count})
	}

	return response
}

/**
 * Create.
 */

func (res *Resource) Create(obj db.Model, user kit.User) kit.Error {
	if allowCreate, ok := res.hooks.(AllowCreateHook); ok {
		if !allowCreate.AllowCreate(res, obj, user) {
			return kit.AppError{Code: "permission_denied"}
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

func (res *Resource) ApiCreate(obj db.Model, r kit.Request) kit.Response {
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

func (res *Resource) Update(obj db.Model, user kit.User) kit.Error {
	oldObj, err := res.FindOne(obj.GetID())
	if err != nil {
		return err
	} else if oldObj == nil {
		return kit.AppError{Code: "not_found"}
	}

	if allowUpdate, ok := res.hooks.(AllowUpdateHook); ok {
		if !allowUpdate.AllowUpdate(res, obj, oldObj, user) {
			return kit.AppError{Code: "permission_denied"}
		}
	}

	if beforeUpdate, ok := res.hooks.(BeforeUpdateHook); ok {
		if err := beforeUpdate.BeforeUpdate(res, obj, oldObj, user); err != nil {
			return err
		}
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

func (res *Resource) ApiUpdate(obj db.Model, r kit.Request) kit.Response {
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

/**
 * Delete.
 */

func (res *Resource) Delete(obj db.Model, user kit.User) kit.Error {
	if allowDelete, ok := res.hooks.(AllowDeleteHook); ok {
		if !allowDelete.AllowDelete(res, obj, user) {
			return kit.AppError{Code: "permission_denied"}
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

func (r ReadOnlyResource) AllowCreate(res kit.Resource, obj db.Model, user kit.User) bool {
	return false
}

func (r ReadOnlyResource) AllowUpdate(res kit.Resource, obj db.Model, user kit.User) bool {
	return false
}

func (r ReadOnlyResource) AllowDelete(res kit.Resource, obj db.Model, user kit.User) bool {
	return false
}
