package cms

import (
	"time"

	db "github.com/theduke/go-dukedb"

	"github.com/theduke/go-appkit/files"
	"github.com/theduke/go-appkit/users"
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

type LocationIntID struct {
	db.IntIDModel
	Location
}

type LocationStrID struct {
	db.StrIDModel
	Location
}

/**
 * Tags.
 */

type BaseTag struct {
	Tag   string `db:"not-null;max:100;index;unique-with:Group"`
	Group string `db:"not-null;max:100;index"`
}

func (t *BaseTag) Collection() string {
	return "tags"
}

type TagStrID struct {
	db.StrIDModel
	BaseTag
}

type TagIntID struct {
	db.IntIDModel
	BaseTag
}

/**
 * Menus.
 */

type MenuItem struct {
	// Internal name for the menu item.
	Name string `db:"not-null;max:200;unique-with:MenuID"`

	// Optinal description for admins.
	Description string

	// Enabled specifies if the item should be shown.
	Enabled bool

	// Whether this item does not actually link to anything but acts as a placeholder.
	IsPlaceholder bool

	// The title that should be rendered.
	Title string `db:"not-null;max:250"`

	// Url this item links to.
	Url string `db:"max:2000"`

	// Route for this item.
	// Use this instead of url if you use a routing system.
	Route string `db:"max:100"`
	// Route arguments, separated by ;.
	RouteArgs string `db:"max:1000"`

	// Used for sorting.
	Weight int `db:"not-null"`
}

func (i *MenuItem) Collection() string {
	return "menu_items"
}

type MenuItemStrID struct {
	db.StrIDModel
	MenuItem

	Menu   *MenuStrID
	MenuID string `db:"max:200;not-null"`

	Parent   *MenuItemStrID
	ParentID string `db:"max:200"`

	Children []*MenuItemStrID `db:"belongs-to:ID:ParentID"`
}

type MenuItemIntID struct {
	db.IntIDModel
	MenuItem

	Menu   *MenuIntID
	MenuID uint64 `db:"not-null"`

	Parent   *MenuItemIntID
	ParentID uint64

	Children []*MenuItemIntID `db:"belongs-to:ID:ParentID"`
}

type Menu struct {
	// Internal identifier for the menu
	Name string `db:"not-null;unique;index;max:200;`

	// Language this menu is in.
	Language string `db:"not-null;min:2;max:2"`

	// Readable title for the menu.
	Title string `db:"not-null;max:250"`

	// Optional menu description.
	Description string `db:"max:500"`
}

func (m *Menu) Collection() string {
	return "menus"
}

type MenuStrID struct {
	db.StrIDModel
	Menu

	TranslatedMenu   *MenuStrID
	TranslatedMenuID string `db:"max:200"`

	Items []*MenuItemStrID `db:"belongs-to:ID:MenuID"`
}

func (i MenuStrID) BeforeDelete(b db.Backend) error {
	// Delete menu items first.
	return b.Q("menu_items").Filter("menu_id", i.ID).Delete()
}

type MenuIntID struct {
	db.IntIDModel
	Menu

	TranslatedMenu   *MenuIntID
	TranslatedMenuID uint64

	Items []*MenuItemIntID `db:"belongs-to:ID:MenuID"`
}

func (i MenuIntID) BeforeDelete(b db.Backend) error {
	// Delete menu items first.
	return b.Q("menu_items").Filter("menu_id", i.ID).Delete()
}

/**
 * Comments.
 */

type Comment struct {
	db.TimeStampedModel
	Type    string `db:"not-null;max:200;"`
	Title   string `db:"max:255"`
	Comment string `db:"not-null"`
}

func (c *Comment) Collection() string {
	return "comments"
}

type CommentStrID struct {
	db.StrIDModel
	Comment
}

type CommentIntID struct {
	db.IntIDModel
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
	Type string `db:"max:255;not-null"`

	// Explicative name for the content.
	Name string `db:"not-null"`

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

type PageComponentStrID struct {
	db.StrIDModel
	PageComponent

	PageID string `db:"not-null"`

	Files []*files.FileStrID `db:"m2m:pages_component_files"`
}

type PageComponentIntID struct {
	db.IntIDModel
	PageComponent

	PageID uint64 `db:"not-null"`

	Files []*files.FileIntID `db:"m2m:pages_component_files"`
}

type Page struct {
	// CreatedAt and UpdatedAt.
	db.TimeStampedModel

	Published   bool
	PublishedAt time.Time

	// Internal title.
	Name string `db:"not-null;max:200"`

	// Type of the page, like "blog post"
	Type string `db:"not-null;max:200"`

	Language string `db:"not-null;min:2;max:2"`

	// Public title.
	Title string `db:"not-null;max:200"`

	// Slug.
	Slug string `db:"not-null;max:250"`

	// Summary for lists.
	ListSummary string

	// Summary for top of the page, seo, etc.
	Summary string

	// Layout to use to display the page.
	Layout string `db:"max:255"`

	// The actual content.
	Content string `db:"not-null;"`
}

func (p Page) Collection() string {
	return "pages"
}

type PageStrID struct {
	db.StrIDModel
	users.StrUserModel
	Page

	MenuItem   *MenuItemStrID
	MenuItemID string `db:"max:200"`

	TranslatedPage   *PageStrID
	TranslatedPageID string `db:"max:200"`

	Files []*files.FileStrID `db:"m2m:pages_files"`

	// Tags.
	Tags []*TagStrID `db:"m2m:pages_tags"`

	// Components.
	Components []*PageComponentStrID `db:"belongs-to:ID:PageID"`
}

func (p PageStrID) BeforeDelete(b db.Backend) error {
	// Delete tags.
	m2m, _ := b.M2M(p, "Tags")
	if err := m2m.Clear(); err != nil {
		return err
	}

	if p.MenuItemID != "" {
		if err := b.Q("menu_items").Filter("id", p.MenuItemID).Delete(); err != nil {
			return err
		}
	}

	return nil
}

type PageIntID struct {
	db.IntIDModel
	users.IntUserModel
	Page

	MenuItem   *MenuItemIntID
	MenuItemID uint64

	TranslatedPage   *PageIntID
	TranslatedPageID string `db:"max:200"`

	Files []*files.FileIntID `db:"m2m:pages_files"`

	// Tags.
	Tags []*TagIntID `db:"m2m:pages_tags"`

	// Components.
	Components []*PageComponentIntID `db:"belongs-to:ID:PageID"`
}

func (p PageIntID) BeforeDelete(b db.Backend) error {
	// Delete tags.
	m2m, _ := b.M2M(p, "Tags")
	if err := m2m.Clear(); err != nil {
		return err
	}

	if p.MenuItemID != 0 {
		if err := b.Q("menu_items").Filter("id", p.MenuItemID).Delete(); err != nil {
			return err
		}
	}

	return nil
}
