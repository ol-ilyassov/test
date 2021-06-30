package data

import (
	"database/sql"
)

type Events struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

type EventModel struct {
	DB *sql.DB
}
