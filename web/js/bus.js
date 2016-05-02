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

    // zoom is the initial zoom value when drawing the Leaflet map
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

    if (this.map == null) {
        this.map = L.map('map').setView([this.lat, this.lon], this.zoom);
    } else {
        this.map.setView([this.lat, this.lon], this.zoom);
    }

    L.tileLayer(this.tileURL, this.tileOptions).addTo(this.map);

    this.getTrips();
};


// appendCell creates a td cell with the value and appends it to row, 
// optionally including an fg and bg color
Bus.prototype.appendCell = function(row, value, fgcolor, bgcolor) {
    var cell = document.createElement("td");
    var cellText = document.createTextNode(value);

    if (fgcolor !== undefined) {
        cell.style.color = fgcolor;
    }

    if (bgcolor !== undefined) {
        cell.style.backgroundColor = bgcolor;
    }

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

// addResult adds a single result value to the page
Bus.prototype.addResult = function(tbody, res) {
    var row = document.createElement("tr");

    // Add the route cell with color
    this.appendCell(
        row, res.stop.route_id,
        "#" + res.route.route_text_color,
        "#" + res.route.route_color
    );

    // Adding the stop name and headsign
    this.appendCell(row, res.stop.stop_name);
    this.appendCell(row, res.stop.headsign);

    // If we have live departures use those, otherwise fall back to
    // scheduled departures
    if (res.departures.live != null && res.departures.live.length > 0) {
        this.appendTime(row, res.departures.live);
    } else {
        this.appendTime(row, res.departures.scheduled);
    }

    // Add cell with distance of the stop from current location
    this.appendCell(row, Math.round(res.dist) + " meters");

    // Append ourselves to the body
    tbody.appendChild(row);
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
        // Parse the response
        var data = JSON.parse(this.response);

        // Destroy the old results if any
        var results = document.getElementById("results");
        if (results.childNodes.length > 0) {
            results.removeChild(results.childNodes[0]);
        }

        // Create a new table with Bootstrap's table class
        var table = document.createElement("table");
        table.setAttribute("class", "table");
        var tbody = document.createElement("tbody");

        // Add each result to our new table
        for (var i = 0; i < data.results.length; i++) {
            self.addResult(tbody, data.results[i]);
        }

        // Display results
        table.appendChild(tbody);
        results.appendChild(table);
    }

    // Trigger the request
    xhr.send();
};
