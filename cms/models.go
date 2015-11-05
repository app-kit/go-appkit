package cms

import (
	"time"

	db "github.com/theduke/go-dukedb"

	"github.com/app-kit/go-appkit/files"
	"github.com/app-kit/go-appkit/users"
)

/**
 * Address.
 */

type Location struct {
	Name        string `db:"max:200"`
	Description string `db:"max:1000"`
	Comments    string `db:"max:1000"`

	Country      string `db:"min:2;max:2;"`
	PostalCode   string `db:"max:100;"`
	State        string `db:"max:100;"`
	Locality     string `db:"max:255;"`
	Street       string `db:"max:500"`
	StreetNumber string `db:"max:100"`
	Top          string `db:"max:100"`

	FormattedAddress string `db:"max:300":`

	Point *db.Point
}

func (a *Location) Collection() string {
	return "locations"
}

type LocationIntId struct {
	db.IntIdModel
	Location
}

type LocationStrId struct {
	db.StrIdModel
	Location
}

/**
 * Tags.
 */

type BaseTag struct {
	Tag   string `db:"required;max:100;index;unique-with:Group"`
	Group string `db:"required;max:100;index"`
}

func (t *BaseTag) Collection() string {
	return "tags"
}

type TagStrId struct {
	db.StrIdModel
	BaseTag
}

type TagIntId struct {
	db.IntIdModel
	BaseTag
}

/**
 * Menus.
 */

type MenuItem struct {
	// Internal name for the menu item.
	Name string `db:"required;max:200;unique-with:MenuId"`

	// Optinal description for admins.
	Description string

	// Enabled specifies if the item should be shown.
	Enabled bool

	// Whether this item does not actually link to anything but acts as a placeholder.
	IsPlaceholder bool

	// The title that should be rendered.
	Title string `db:"required;max:250"`

	// Url this item links to.
	Url string `db:"max:2000"`

	// Route for this item.
	// Use this instead of url if you use a routing system.
	Route string `db:"max:100"`
	// Route arguments, separated by ;.
	RouteArgs string `db:"max:1000"`

	// Used for sorting.
	Weight int `db:"required"`
}

func (i *MenuItem) Collection() string {
	return "menu_items"
}

type MenuItemStrId struct {
	db.StrIdModel
	MenuItem

	Menu   *MenuStrId
	MenuId string `db:"max:200;required"`

	Parent   *MenuItemStrId
	ParentId string `db:"max:200"`

	Children []*MenuItemStrId `db:"belongs-to:Id:ParentId"`
}

type MenuItemIntId struct {
	db.IntIdModel
	MenuItem

	Menu   *MenuIntId
	MenuId uint64 `db:"required"`

	Parent   *MenuItemIntId
	ParentId uint64

	Children []*MenuItemIntId `db:"belongs-to:Id:ParentId"`
}

type Menu struct {
	// Internal identifier for the menu
	Name string `db:"required;unique;index;max:200;`

	// Language this menu is in.
	Language string `db:"required;min:2;max:2"`

	// Readable title for the menu.
	Title string `db:"required;max:250"`

	// Optional menu description.
	Description string `db:"max:500"`
}

func (m *Menu) Collection() string {
	return "menus"
}

type MenuStrId struct {
	db.StrIdModel
	Menu

	TranslatedMenu   *MenuStrId
	TranslatedMenuId string `db:"max:200"`

	Items []*MenuItemStrId `db:"belongs-to:Id:MenuId"`
}

func (i MenuStrId) BeforeDelete(b db.Backend) error {
	// Delete menu items first.
	return b.Q("menu_items").Filter("menu_id", i.Id).Delete()
}

type MenuIntId struct {
	db.IntIdModel
	Menu

	TranslatedMenu   *MenuIntId
	TranslatedMenuId uint64

	Items []*MenuItemIntId `db:"belongs-to:Id:MenuId"`
}

func (i MenuIntId) BeforeDelete(b db.Backend) error {
	// Delete menu items first.
	return b.Q("menu_items").Filter("menu_id", i.Id).Delete()
}

/**
 * Comments.
 */

type Comment struct {
	db.TimeStampedModel
	Type    string `db:"required;max:200;"`
	Title   string `db:"max:255"`
	Comment string `db:"required"`
}

func (c *Comment) Collection() string {
	return "comments"
}

type CommentStrId struct {
	db.StrIdModel
	Comment
}

type CommentIntId struct {
	db.IntIdModel
	Comment
}

/**
 * Pages.
 */

/**
 * Components.
 */

type PageComponent struct {
	// Component type.
	Type string `db:"max:255;required"`

	// Explicative name for the content.
	Name string `db:"required"`

	// Data for the component.
	Data string `db:""`

	// Weight for sorting.
	Weight int

	// Whether to show the component.
	Enabled bool
}

func (PageComponent) Collection() string {
	return "page_components"
}

type PageComponentStrId struct {
	db.StrIdModel
	PageComponent

	PageId string `db:"required"`

	Files []*files.FileStrId `db:"m2m:pages_component_files"`
}

type PageComponentIntId struct {
	db.IntIdModel
	PageComponent

	PageId uint64 `db:"required"`

	Files []*files.FileIntId `db:"m2m:pages_component_files"`
}

type Page struct {
	// CreatedAt and UpdatedAt.
	db.TimeStampedModel

	Published   bool
	PublishedAt time.Time

	// Internal title.
	Name string `db:"required;max:200"`

	// Type of the page, like "blog post"
	Type string `db:"required;max:200"`

	Language string `db:"required;min:2;max:2"`

	// Public title.
	Title string `db:"required;max:200"`

	// Slug.
	Slug string `db:"required;max:250"`

	// Summary for lists.
	ListSummary string

	// Summary for top of the page, seo, etc.
	Summary string

	// Layout to use to display the page.
	Layout string `db:"max:255"`

	// The actual content.
	Content string `db:"required;"`
}

func (p Page) Collection() string {
	return "pages"
}

type PageStrId struct {
	db.StrIdModel
	users.StrUserModel
	Page

	MenuItem   *MenuItemStrId
	MenuItemId string `db:"max:200"`

	TranslatedPage   *PageStrId
	TranslatedPageId string `db:"max:200"`

	Files []*files.FileStrId `db:"m2m:pages_files"`

	// Tags.
	Tags []*TagStrId `db:"m2m:pages_tags"`

	// Components.
	Components []*PageComponentStrId `db:"belongs-to:Id:PageId"`
}

func (p PageStrId) BeforeDelete(b db.Backend) error {
	// Delete tags.
	m2m, _ := b.M2M(p, "Tags")
	if err := m2m.Clear(); err != nil {
		return err
	}

	if p.MenuItemId != "" {
		if err := b.Q("menu_items").Filter("id", p.MenuItemId).Delete(); err != nil {
			return err
		}
	}

	return nil
}

type PageIntId struct {
	db.IntIdModel
	users.IntUserModel
	Page

	MenuItem   *MenuItemIntId
	MenuItemId uint64

	TranslatedPage   *PageIntId
	TranslatedPageId string `db:"max:200"`

	Files []*files.FileIntId `db:"m2m:pages_files"`

	// Tags.
	Tags []*TagIntId `db:"m2m:pages_tags"`

	// Components.
	Components []*PageComponentIntId `db:"belongs-to:Id:PageId"`
}

func (p PageIntId) BeforeDelete(b db.Backend) error {
	// Delete tags.
	m2m, _ := b.M2M(p, "Tags")
	if err := m2m.Clear(); err != nil {
		return err
	}

	if p.MenuItemId != 0 {
		if err := b.Q("menu_items").Filter("id", p.MenuItemId).Delete(); err != nil {
			return err
		}
	}

	return nil
}
