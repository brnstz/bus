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
    this.tileURL = 'http://otile1.mqcdn.com/tiles/1.0.0/map/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    this.tileOptions = {
        MaxZoom: 20
    };

    // zoom is the zoom value when drawing the Leaflet map
    this.zoom = 15;

    // map is our Leaflet JS map object
    this.map = null;
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
    this.lat = position.coords.latitude;
    this.lon = position.coords.longitude;

    this.map = L.map('map').setView([this.lat, this.lon], this.zoom);
    L.tileLayer(this.tileURL, this.tileOptions).addTo(this.map);

    this.getTrips();
};


// appendCell creates a td cell with the value and appends it to row
Bus.prototype.appendCell = function(row, value) {
    var cell = document.createElement("td");
    var cellText = document.createTextNode(value);

    cell.appendChild(cellText);
    row.appendChild(cell);
};

// appendCell creates a td cell with the value and appends it to row
// along with setting the fg and bg colors
Bus.prototype.appendCellColor = function(row, value, fgcolor, bgcolor) {
    var cell = document.createElement("td");
    var cellText = document.createTextNode(value);

    cell.style.color = "#" + fgcolor;
    cell.style.backgroundColor = "#" + bgcolor;

    cell.appendChild(cellText);
    row.appendChild(cell);
};

// hourFormat takes a time object and returns the format we want to display on
// screen
Bus.prototype.timeFormat = function(time) {
    var t = new Date(time);

    // Get minutes as a 00 padded value
    var minutes = ("00" + t.getMinutes()).slice(-2);

    // Get a temporary value for hours, format it below
    var hours = t.getHours();

    // Convert to US time
    if (hours > 12) {
        hours -= 12;
    }
    if (hours == 0) {
        hours = 12;
    }

    return hours + ":" + minutes;
}

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

// getTrips calls the stops API with our current state and updates
// the UI with the results
Bus.prototype.getTrips = function() {
    var self = this;
    var xhr = new XMLHttpRequest();
    var url = '/api/v2/stops?lat=' + this.lat +
        '&lon=' + this.lon +
        '&filter=' + this.filter +
        '&miles=' + this.miles;

    xhr.open('GET', url);
    xhr.onload = function(e) {
        var data = JSON.parse(this.response);
        var tbl = document.createElement("table");
        tbl.setAttribute("class", "table");
        var tblBody = document.createElement("tbody");
        var results = document.getElementById("results");

        if (results.childNodes.length > 0) {
            results.removeChild(results.childNodes[0]);
        }

        for (var i = 0; i < data.results.length; i++) {
            var res = data.results[i];
            var row = document.createElement("tr");

            self.appendCellColor(row, res.stop.route_id, res.route.route_text_color, res.route.route_color);
            self.appendCell(row, res.stop.stop_name);
            self.appendCell(row, res.stop.headsign);

            if (res.departures.live != null && res.departures.live.length > 0) {
                self.appendTime(row, res.departures.live);
            } else {
                self.appendTime(row, res.departures.scheduled);
            }

            self.appendCell(row, Math.round(res.dist) + " meters");

            tblBody.appendChild(row);
        }

        tbl.appendChild(tblBody);
        results.appendChild(tbl);
    }
    xhr.send();
};
