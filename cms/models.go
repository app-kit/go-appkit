package cms

import (
	db "github.com/theduke/go-dukedb"

	"github.com/theduke/go-appkit/files"
	"github.com/theduke/go-appkit/users"
)

/**
 * Address.
 */

type Address struct {
	Country      string `db:"min:2;max:2;"`
	PostalCode   string `db:"max:100;"`
	State        string `db:"max:100;"`
	Locality     string `db:"max:255;"`
	Street       string `db:"max:500"`
	StreetNumber string `db:"max:100"`
	Top          string `db:"max:100"`

	Description string `db:"max:1000"`

	Latitude  string `db:"max:100"`
	Longitude string `db:"max:100"`
}

func (a *Address) Collection() string {
	return "addresses"
}

type AddressIntID struct {
	db.IntIDModel
	Address
}

type AddressStrID struct {
	db.StrIDModel
	Address
}

/**
 * Tags.
 */

type Tag struct {
	Tag  string `db:"primary-key;not-null;max:100;index;unique-with:Type"`
	Type string `db:"not-null;max:100;index"`
}

func (t *Tag) Collection() string {
	return "tags"
}

func (t *Tag) GetID() interface{} {
	return t.Tag
}

func (t *Tag) SetID(tag interface{}) error {
	t.Tag = tag.(string)
	return nil
}

func (t *Tag) GetStrID() string {
	return t.Tag
}

func (t *Tag) SetStrID(tag string) error {
	t.Tag = tag
	return nil
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
}

type MenuIntID struct {
	db.IntIDModel
	Menu

	TranslatedMenu   *MenuIntID
	TranslatedMenuID uint64
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

type Page struct {
	// CreatedAt and UpdatedAt.
	db.TimeStampedModel

	Published bool

	// Internal title.
	Name string `db:"not-null;max:200"`

	// Type of the page, like "blog post"
	Type string `db:"not-null;max:200"`

	Language string `db:"not-null;min:2;max:2"`

	// Public title.
	Title string `db:"not-null;max:200"`

	// Slug.
	Slug string `db:"not-null;max:250"`

	// Short summary for lists, SEO, etc..
	Summary string

	// The actual content.
	Content string `db:"not-null;"`

	// Tags.
	Tags []*Tag `db:"m2m:pages_tags"`

	Comments []*Comment `db:"m2m:pages_comments"`
}

func (p Page) Collection() string {
	return "pages"
}

type PageStrID struct {
	db.StrIDModel
	users.StrUserModel
	Page

	MenuItem   *MenuStrID
	MenuItemID string `db:"max:200"`

	TranslatedPage   *PageStrID
	TranslatedPageID string `db:"max:200"`

	Files         []*files.FileStrID `db:"m2m:pages_files"`
	AttachedFiles []*files.FileStrID `db:"m2m:pages_attached_files"`
}

type PageIntID struct {
	db.IntIDModel
	users.IntUserModel
	Page

	MenuItem   *MenuIntID
	MenuItemID string `db:"max:200"`

	TranslatedPage   *PageIntID
	TranslatedPageID string `db:"max:200"`

	Files         []*files.FileIntID `db:"m2m:pages_files"`
	AttachedFiles []*files.FileIntID `db:"m2m:pages_attached_files"`
}
