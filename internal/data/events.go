package data

import (
	"database/sql"
	"time"
)

type Events struct {
	EventId      int64      `json:"event_id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	IconId       int64      `json:"icon_id"`
	ContactsLink int64      `json:"contacts_link"`
	CreatedTime  *time.Time `json:"created_time"`
}

type EventModel struct {
	DB *sql.DB
}
