package data

import (
	"database/sql"
	"time"
)

type User struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"` // !
	Activated bool      `json:"activated"`
	Version   int       `json:"-"` // !
}

type password struct {
	plaintext *string // distinguish between not being present versus empty string
	hash      []byte
}

type UserModel struct {
	DB *sql.DB
}
