// bus is our controller for the bus application. It handles AJAX requests, 
// drawing to the screen and creating/managing other objects.
var bus = new Bus();

function Bus() {
    // lat, lon is the center of our request. We send this to the Bus API
    // and also use it to draw the map. We can get this value from the
    // HTML5 location API.
    this.lat = 0;
    this.lon = 0;

    // miles and filter are options sent to the Bus API
    this.miles = 0.5;
    this.filter = '';

    // tileURL is passed to Leaflet JS for drawing the map
    //this.tileURL = 'https://otile1-s.mqcdn.com/tiles/1.0.0/map/{z}/{x}/{y}.png';
    this.tileURL = 'https://stamen-tiles.a.ssl.fastly.net/toner-lite/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    this.tileOptions = {
        MaxZoom: 20
    };

    // zoom is the initial zoom value when drawing the Leaflet map
    this.zoom = 16;

    // map is our Leaflet JS map object
    this.map = null;

    // resultsMap is the most recent list of results from the API, mapped
    // from ID to a Result object.
    this.resultsMap = {};

    // results is the list of results in the order returned by the API 
    // (i.e., distance from location)
    this.results = [];
}

// init is run when the page initially loads
Bus.prototype.init = function() {
    this.refresh();
};

// refresh requests the location from the browser, sets our lat / lon and
// gets new trips from the API 
Bus.prototype.refresh = function() {
    var self = this;

    if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(function(p) {
            self.updatePosition(p)
        });
    }
};

// updatePosition takes an HTML5 geolocation position response and 
// updates our map and trip info
Bus.prototype.updatePosition = function(position) {

    // Set our lat and lon based on the coords
    this.lat = position.coords.latitude;
    this.lon = position.coords.longitude;

    // If we don't have a map, create one.
    if (this.map == null) {
        this.map = L.map('map');
    }

    // Set location and zoom of the map.
    this.map.setView([this.lat, this.lon], this.zoom);

    // Add our tiles
    L.tileLayer(this.tileURL, this.tileOptions).addTo(this.map);

    // Get the results for this location
    this.getTrips();
};


// appendCell creates a td cell with the value and appends it to row, 
// optionally including an fg and bg color
Bus.prototype.appendCell = function(row, value, fgcolor, bgcolor) {

    // Create the cell and its text
    var cell = document.createElement("td");
    var cellText = document.createTextNode(value);

    // Set colors when requested
    if (fgcolor !== undefined) {
        cell.style.color = fgcolor;
    }
    if (bgcolor !== undefined) {
        cell.style.backgroundColor = bgcolor;
    }

    // Add values to the actual row
    cell.appendChild(cellText);
    row.appendChild(cell);
};


// appendTime adds a cell to this row with the current time values
Bus.prototype.appendTime = function(row, departures) {
    var mytext = "";

    if (departures != null) {
        for (var i = 0; i < departures.length; i++) {
            mytext += " " + this.timeFormat(departures[i].time);
        }
    }

    this.appendCell(row, mytext);
};

// draw puts the current state of bus onto the screen
Bus.prototype.draw = function() {
    var self = this;

    // Destroy the old resDiv if any
    var resDiv = document.getElementById("results");
    if (resDiv.childNodes.length > 0) {
        resDiv.removeChild(resDiv.childNodes[0]);
    }

    // Create a new table with Bootstrap's table class and also
    // the tbody
    var table = document.createElement("table");
    table.setAttribute("class", "table");
    var tbody = document.createElement("tbody");

    // Add each result to our new table
    for (var i = 0; i < self.results.length; i++) {
        var r = self.results[i];

        tbody.appendChild(r.row);

        // Put it on the map
        r.marker.addTo(self.map);
    }

    // Display results
    table.appendChild(tbody);
    results.appendChild(table);
};

Bus.prototype.clickResult = function(res) {
    var self = this;
    console.log("here is res", res);

    // Set all results to background
    for (var i = 0; i < self.results.length; i++) {
        self.results[i].background();
    }

    // Set this one to foreground
    self.resultsMap[res.result.id].foreground();

    // Re-center map on this result
    self.map.setView([res.result.stop.lat, res.result.stop.lon]);
};

// getTrips calls the stops API with our current state and updates
// the UI with the results
Bus.prototype.getTrips = function() {
    var self = this;

    // Create an AJAX request with our current location
    var xhr = new XMLHttpRequest();
    var url = '/api/v2/stops?lat=' + this.lat +
        '&lon=' + this.lon +
        '&filter=' + this.filter +
        '&miles=' + this.miles;

    // Open the connection
    xhr.open('GET', url);

    // When it succeeds, update the page
    xhr.onload = function(e) {
        console.log("onload says", e);

        // Parse the response
        var data = JSON.parse(this.response);

        // Reset stops value
        self.results = [];
        self.resultsMap = {};

        // Add each result to our list
        for (var i = 0; i < data.results.length; i++) {
            var r = new Result(data.results[i]);
            self.results[i] = r;
            self.resultsMap[r.result.id] = r;

            // Set onclick for the result
            r.row.onclick = function() {
                console.log("here is this1", this);
                self.clickResult(r);
            }
            r.marker.onclick = function() {
                console.log("here is this2", this);
                self.clickResult(r);
            }
        }

        self.draw();
    }

    // Trigger the request
    xhr.send();
};
