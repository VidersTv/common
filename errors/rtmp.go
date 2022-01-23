package errors

import "fmt"

var (
	ErrInvalidStreamKey = fmt.Errorf("invalid streamkey")
	ErrAlreadyAuthed    = fmt.Errorf("already authed")
)
