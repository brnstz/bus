package api

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"path"
	"strconv"

	"github.com/julienschmidt/httprouter"

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
	// Create our mux
	mux := httprouter.New()

	// Set up index and dynamic endpoints
	mux.GET("/", getIndex)
	mux.GET("/api/stops", getStops)
	mux.GET("/api/routes", getRoutes)
	mux.GET("/api/agencies/:agencyID/routes/:routeID/trips/:tripID", getTrip)

	// Create static endpoints
	for _, v := range staticPaths {
		endpoint := "/" + v + "/*filepath"
		dir := http.Dir(path.Join(conf.API.WebDir, v))
		mux.ServeFiles(endpoint, dir)
	}

	return mux
}

func getIndex(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	// Create index template
	indexTemplate, err := template.ParseFiles(
		path.Join(conf.API.WebDir, "index.html"),
	)
	if err != nil {
		apiErr(w, err)
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
