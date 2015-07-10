package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/brnstz/bus/common"
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

func getUI(w http.ResponseWriter, r *http.Request) {
	ui := []byte(`
		<!DOCTYPE html>
		<html>
		<body>


		<script>
			var x = document.getElementById("demo");

			function getLocation() {
				if (navigator.geolocation) {
					navigator.geolocation.getCurrentPosition(showPosition);
				}
			}

			function showPosition(position) {
				document.getElementById("lat").setAttribute("value", position.coords.latitude);
				document.getElementById("lon").setAttribute("value", position.coords.longitude);
			}

			function setLocation(lat, lon, miles) {
				document.getElementById("lat").setAttribute("value", lat);
				document.getElementById("lon").setAttribute("value", lon);
				document.getElementById("miles").setAttribute("value", miles);
			}

			function getTrips() {
				var xhr = new XMLHttpRequest();
				var url = '/api/v1/stops?lat=' + document.getElementById("lat").value +
						  '&lon='			   + document.getElementById("lon").value +
						  '&filter='	       + document.getElementById("filter").value +
						  '&miles='	           + document.getElementById("miles").value;

				xhr.open('GET', url);
				xhr.onload = function(e) {
					  var data = JSON.parse(this.response);
					  console.log(data);
				}
				xhr.send();
			}

		</script>

		Latitude: <input type="text" id="lat" name="lat"><br>
		Longitude: <input type="text" id="lon" name="lon"><br>
		Filter:
			<select id="filter">
				<option value="">Subway and bus</option>
				<option value="subway">Subway only</option>
				<option value="bus">Bus only</option>
			</select><br>
		Radius: <input type="text" id="miles" value="0.2"> miles<br>

		<button onclick="getLocation()">Detect location</button><br>
		<button onclick="setLocation(40.758895,-73.985131, 0.2)">Times Square</button><br>
		<button onclick="setLocation(40.7236448,-74.0006793, 0.2)">SoHo</button><br>
		<button onclick="setLocation(40.7293373,-73.9458161, 0.2)">Greenpoint</button><br>
		<button onclick="setLocation(40.6825236,-73.9750134, 0.2)">Barclays Center</button><br>
		<button onclick="setLocation(40.84932,-73.877154,15, 0.2)">Bronx Zoo</button><br>
		<button onclick="setLocation(40.7501217,-73.8463344, 0.3)">US Open</button><br>
		<button onclick="setLocation(40.5031274,-74.253251, 0.3)">Conference House Park</button><br><br>


		<button onclick="getTrips()">Get upcoming trips</button><br>

		</body>
		</html>
	`)
	w.Write(ui)
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime)

	//go loader.LoadForever()

	http.HandleFunc("/api/v1/stops", getStops)
	http.HandleFunc("/", getUI)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", os.Getenv("BUS_API_PORT")), nil))

}
