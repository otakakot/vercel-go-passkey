package api

import (
	_ "embed"
	"net/http"
)

//go:embed _index.html
var yaml []byte

func Handler(
	rw http.ResponseWriter,
	req *http.Request,
) {
	rw.Write(yaml)
}
