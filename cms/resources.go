package cms

import (
	"time"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/app/methods"
	"github.com/theduke/go-appkit/resources"
	"github.com/theduke/go-appkit/utils"
)

/*
	db "github.com/theduke/go-dukedb"
	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/files"
*/

type MenuResource struct {
	resources.AdminResource
}

type MenuItemResource struct {
	resources.AdminResource
}

type CommentResource struct {
}

type PageResource struct {
	resources.UserResource
}

func (PageResource) Methods(res kit.Resource) []kit.Method {

	publish := &methods.Method{
		Name:     "cms.page.publish",
		Blocking: true,
		Handler: func(registry kit.Registry, r kit.Request, unblock func()) kit.Response {
			user := r.GetUser()
			if user == nil || !user.HasRole("admin") {
				return kit.NewErrorResponse("permission_denied")
			}

			id := utils.GetMapStringKey(r.GetData(), "id")

			if id == "" {
				return kit.NewErrorResponse("no_id_in_data", "Expected 'id' key in data.")
			}

			rawPage, err := res.Backend().FindOne("pages", id)
			if err != nil {
				return kit.NewErrorResponse("db_error", err)
			} else if rawPage == nil {
				return kit.NewErrorResponse("not_found", "The specified page id does not exist.")
			}

			err = res.Backend().UpdateByMap(rawPage, map[string]interface{}{
				"published":    true,
				"published_at": time.Now(),
			})

			if err != nil {
				return kit.NewErrorResponse("db_error", err)
			}

			return &kit.AppResponse{
				Data: map[string]interface{}{"success": true},
			}
		},
	}

	return []kit.Method{publish}
}

func (PageResource) AllowFind(res kit.Resource, obj kit.Model, user kit.User) bool {
	if p, ok := obj.(*PageIntID); ok && p.Published {
		return true
	} else if p, ok := obj.(*PageStrID); ok && p.Published {
		return true
	}

	u := obj.(kit.UserModel)

	return user != nil && (u.GetUserID() == user.GetID() || user.HasRole("admin"))
}

func (PageResource) BeforeUpdate(res kit.Resource, obj, oldobj kit.Model, user kit.User) apperror.Error {
	return nil
}

func (PageResource) BeforeDelete(res kit.Resource, obj kit.Model, user kit.User) apperror.Error {
	// Note: other relationship deletion handling happens in models BeforeDelete() hook.
	// Files have to be deleted here, since they require the fileService.

	// Delete files.
	fileService := res.Registry().FileService()

	m2m, err := res.Backend().M2M(obj, "Files")
	if err != nil {
		return err
	}

	files := m2m.All()

	// Delete m2m relation.
	if err := m2m.Clear(); err != nil {
		return err
	}

	for _, file := range files {
		if err := fileService.Delete(file.(kit.File), user); err != nil {
			return err
		}
	}

	return nil
}

func (PageResource) ApiAlterQuery(res kit.Resource, query db.Query, r kit.Request) apperror.Error {
	typ := r.GetContext().String("type")
	if typ != "" {
		query.Filter("type", typ)
	}
	return nil
}
