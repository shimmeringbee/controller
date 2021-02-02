package swagger

import (
	"embed"
	"github.com/gorilla/mux"
	"io/fs"
	"net/http"
)

//go:embed dist/swagger-ui/dist/*
var swagger embed.FS

//go:embed dist/index/*
var index embed.FS

func mustSub(parentFS fs.FS, prefix string) fs.FS {
	subFs, err := fs.Sub(parentFS, prefix)
	if err != nil {
		panic(err)
	}

	return subFs
}

func ConstructRouter() http.Handler {
	r := mux.NewRouter()
	// This route overrides the swagger-ui distribution for index.html, http.FileServer will not service index.html
	// by name, but instead servers it on the root path. The PathPrefix below provides a redirect from index.html -> /,
	// if needed.
	r.Path("/").Handler(http.FileServer(http.FS(mustSub(index, "dist/index")))).Methods("GET")
	// Server all swagger-ui assets (other than index.html) from the swagger embedded file system.
	r.PathPrefix("/").Handler(http.FileServer(http.FS(mustSub(swagger, "dist/swagger-ui/dist")))).Methods("GET")

	return r
}
