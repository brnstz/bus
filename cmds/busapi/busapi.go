package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/kelseyhightower/envconfig"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

func floatOrDie(w http.ResponseWriter, r *http.Request, name string) (f float64, err error) {

	val := r.FormValue(name)
	f, err = strconv.ParseFloat(val, 64)
	if err != nil {
		log.Println("bad float value", val, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	return
}

func getStops(w http.ResponseWriter, r *http.Request) {
	lat, err := floatOrDie(w, r, "lat")
	if err != nil {
		return
	}

	lon, err := floatOrDie(w, r, "lon")
	if err != nil {
		return
	}

	miles, err := floatOrDie(w, r, "miles")
	if err != nil {
		return
	}

	filter := r.FormValue("filter")

	meters := etc.MileToMeter(miles)

	stops, err := models.GetStopsByLoc(etc.DBConn, lat, lon, meters, filter)
	if err != nil {
		log.Println("can't get stops", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	b, err := json.Marshal(stops)
	if err != nil {
		log.Println("can't marshal to json", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

func main() {
	var err error
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	err = envconfig.Process("bus", &conf.DB)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.API)
	if err != nil {
		log.Fatal(err)
	}

	etc.DBConn = etc.MustDB()

	http.HandleFunc("/api/v1/stops", getStops)
	http.Handle("/", http.FileServer(http.Dir(conf.API.WebDir)))

	log.Fatal(http.ListenAndServe(conf.API.Addr, nil))
}
