package dto

// Item represents a simple entity used for demo CRUD operations.
type Item struct {
	ID   int    `json:"id" example:"1"`
	Name string `json:"name" example:"\"sample\""`
}
