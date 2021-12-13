package pprof

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/shimmeringbee/controller/interface/http/auth"
	"net/http"
	"net/http/pprof"
)

func ConstructRouter(ap auth.AuthenticationProvider) http.Handler {
	pprofRoute := mux.NewRouter()

	pprofRoute.PathPrefix("/cmdline").HandlerFunc(pprof.Cmdline)
	pprofRoute.PathPrefix("/profile").HandlerFunc(pprof.Profile)
	pprofRoute.PathPrefix("/symbol").HandlerFunc(pprof.Symbol)
	pprofRoute.PathPrefix("/trace").HandlerFunc(pprof.Trace)
	pprofRoute.PathPrefix("/").Handler(http.StripPrefix("/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "" {
			request.URL.Path = fmt.Sprintf("/debug/pprof/%s", request.URL.Path)
		}

		pprof.Index(writer, request)
	})))

	return ap.AuthenticationMiddleware(pprofRoute)
}
