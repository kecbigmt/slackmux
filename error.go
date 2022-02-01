package slackmux

import (
	"errors"
	"net/http"
)

var (
	ErrCommandNotFound = errors.New("command not found")
)

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, error)
