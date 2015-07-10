package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/brnstz/bus/common"
	"github.com/brnstz/bus/loader"
	"github.com/brnstz/bus/models"
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

	meters := common.MileToMeter(miles)

	stops, err := models.GetStopsByLoc(common.DB, lat, lon, meters, filter)
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
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	go loader.LoadForever()

	http.HandleFunc("/api/v1/stops", getStops)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", os.Getenv("BUS_API_PORT")), nil))

}
