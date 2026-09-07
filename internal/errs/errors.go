package errs

import (
	"errors"
)

var (
	ErrInvalidID = errors.New("mongostore: invalid lock id")
	ErrNotFound  = errors.New("mongostore: lock not found")
)