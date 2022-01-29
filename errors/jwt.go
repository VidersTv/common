package errors

import (
	"fmt"
)

var (
	ErrJwtTokenInvalid = fmt.Errorf("jwt token is invalid")
)
