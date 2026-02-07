package model

import "time"

// BaseModel provides common fields for all database models
type BaseModel struct {
	ID                 string    `json:"id" firestore:"id" db:"id"`
	Active             bool      `json:"active" firestore:"active" db:"active"`
	DateCreated        time.Time `json:"date_created" firestore:"date_created" db:"date_created"`
	DateCreatedString  string    `json:"date_created_string" firestore:"date_created_string" db:"date_created_string"`
	DateModified       time.Time `json:"date_modified" firestore:"date_modified" db:"date_modified"`
	DateModifiedString string    `json:"date_modified_string" firestore:"date_modified_string" db:"date_modified_string"`
}

// SetCreateProperties sets properties for new records
func (b *BaseModel) SetCreateProperties(id string) {
	now := time.Now().UTC()
	b.ID = id
	b.Active = true
	b.DateCreated = now
	b.DateCreatedString = now.Format(time.RFC3339)
	b.DateModified = now
	b.DateModifiedString = now.Format(time.RFC3339)
}

// SetUpdateProperties sets properties for updated records
func (b *BaseModel) SetUpdateProperties() {
	now := time.Now().UTC()
	b.DateModified = now
	b.DateModifiedString = now.Format(time.RFC3339)
}

// SetDeleteProperties sets properties for soft-deleted records
func (b *BaseModel) SetDeleteProperties() {
	now := time.Now().UTC()
	b.Active = false
	b.DateModified = now
	b.DateModifiedString = now.Format(time.RFC3339)
}
