package dto

// CreateItemInput defines the payload for creating an item.
type CreateItemInput struct {
	Body Item `body:""`
}

// IDPath holds a path ID parameter.
type IDPath struct {
	ID int `path:"id" example:"1"`
}

// ListItemsInput captures optional filters used when listing items. Fields
// tagged with `internal:"true"` are omitted from public documentation.
type ListItemsInput struct {
	Debug string `query:"debug" internal:"true"`
	Token string `header:"X-Debug" internal:"true"`
	Limit int    `query:"limit" minimum:"1" maximum:"100" example:"10"`
}

// UpdateItemInput defines the payload for updating an item.
type UpdateItemInput struct {
	ID   int  `path:"id" example:"1"`
	Body Item `body:""`
}
