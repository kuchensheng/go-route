package main

import (
	"net/http"
)

type Plugin interface {
	Handler(req *http.Request, target []byte) error
}
