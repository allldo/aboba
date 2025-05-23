package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type MangaStatus string

const (
	StatusOngoing   MangaStatus = "ongoing"
	StatusCompleted MangaStatus = "completed"
	StatusAnnounced MangaStatus = "announced"
	StatusCancelled MangaStatus = "cancelled"
)

type StringArray []string

func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = StringArray{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, sa)
	case string:
		return json.Unmarshal([]byte(v), sa)
	default:
		return errors.New("cannot scan into StringArray")
	}
}

func (sa StringArray) Value() (driver.Value, error) {
	return json.Marshal(sa)
}

type Manga struct {
	ID          int64       `db:"id" json:"id"`
	Title       string      `db:"title" json:"title"`
	Description string      `db:"description" json:"description"`
	Author      string      `db:"author" json:"author"`
	Artist      string      `db:"artist" json:"artist"`
	Genres      StringArray `db:"genres" json:"genres"`
	Status      MangaStatus `db:"status" json:"status"`
	Year        int         `db:"year" json:"year"`
	Chapters    int         `db:"chapters" json:"chapters"`
	Price       float64     `db:"price" json:"price"`
	CoverImage  string      `db:"cover_image" json:"cover_image"`
	Stock       int         `db:"stock" json:"stock"`
	IsActive    bool        `db:"is_active" json:"is_active"`
	CreatedAt   string      `db:"created_at" json:"created_at"`
	UpdatedAt   string      `db:"updated_at" json:"updated_at"`
}
