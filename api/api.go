package api

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/models"
)

var (
	errBadRequest = errors.New("bad request")

	// errCodes is a mapping from known errors to HTTP status codes
	errCodes = map[error]int{
		models.ErrNotFound:         http.StatusNotFound,
		models.ErrInvalidRouteType: http.StatusBadRequest,
		errBadRequest:              http.StatusBadRequest,
	}

	staticPaths = []string{"js", "css", "img"}
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	// Get stop / route / trip info close to here
	mux.HandleFunc("/api/here", getHere)

	// Get routes that should be preloaded
	mux.HandleFunc("/api/routes", getRoutes)

	// Get a single route by agency_id / route_id
	mux.HandleFunc("/api/route", getRoute)

	// Get a single trip by agency_id / route_id / trip_id
	mux.HandleFunc("/api/trip", getTrip)

	// Add specific handlers for each static directory. These will
	// be served directly.
	for _, v := range staticPaths {
		pattern := "/" + v + "/"
		dir := path.Join(conf.API.WebDir, v)
		fs := http.StripPrefix(pattern, http.FileServer(http.Dir(dir)))
		mux.Handle(pattern, fs)
	}

	// The index template gets special treatment to add the build
	// timestamp and also to possibly allow different caching treatment.
	mux.HandleFunc("/", getIndex)

	return mux
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.RequestURI)
	if err != nil {
		apiErr(w, err)
		return
	}

	switch u.Path {
	case "/", "/index.html":

	default:
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Create index template
	indexTemplate, err := template.ParseFiles(
		path.Join(conf.API.WebDir, "index.html"),
	)
	if err != nil {
		apiErr(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	indexTemplate.Execute(w, map[string]interface{}{
		"BuildTimestamp": conf.API.BuildTimestamp,
	})
}

func floatOrDie(val string) (f float64, err error) {

	f, err = strconv.ParseFloat(val, 64)
	if err != nil {
		log.Println("bad float value", val, err)
		err = errBadRequest
		return
	}

	return
}

// apiErr writes an appropriate response to w given the incoming error
// by looking at the errCodes map
func apiErr(w http.ResponseWriter, err error) {
	code, ok := errCodes[err]
	if !ok {
		code = http.StatusInternalServerError
	}

	w.WriteHeader(code)
}
