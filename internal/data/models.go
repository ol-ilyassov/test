package data

import (
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Users UserModel
}

func NewModels() Models {
	return Models{
		Users: UserModel{},
	}
}

//func NewModels(db *sql.DB) Models {
//	return Models{
//		Users: UserModel{},
//	}
//}
