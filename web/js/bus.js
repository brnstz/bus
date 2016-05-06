// bus is our controller for the bus application. It handles drawing to the
// screen and managing objects.
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
    this.tileURL = 'https://stamen-tiles.a.ssl.fastly.net/toner-lite/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    this.tileOptions = {
        MaxZoom: 20
    };

    // zoom is the initial zoom value when drawing the Leaflet map
    this.zoom = 16;

    // map is our Leaflet JS map object
    this.map = null;

    // stopList is the list of results in the order returned by the API 
    // (i.e., distance from location)
    this.stopList = [];

    // stops is stop ids mapped to stop objects
    this.stops = {};

    // markers is stop ids mapped to markers on the map
    this.markers = {};

    // rows is stop ids mapped to rows in the results table
    this.rows = {};
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
    var self = this;

    // Set our lat and lon based on the coords
    self.lat = position.coords.latitude;
    self.lon = position.coords.longitude;

    // If we don't have a map, create one.
    if (self.map == null) {
        self.map = L.map('map');
    }

    // Set location and zoom of the map.
    self.map.setView([self.lat, self.lon], self.zoom);

    // Add our tiles
    L.tileLayer(self.tileURL, self.tileOptions).addTo(self.map);

    // Get the results for this location
    self.getStops();
};



// parseStops reads the text of response from the stops API and updates
// the initial list of stop objects
Bus.prototype.parseStops = function(data) {
    var self = this;

    // Reset list of stops
    self.stopList = [];

    // Create a stop object for each result and save to our list
    for (var i = 0; i < data.results.length; i++) {
        var s = new Stop(data.results[i]);
        self.stopList[i] = s;
    }
};

Bus.prototype.createMarker = function(stop) {
    var self = this;

    return L.circle(
        [stop.api.stop.lat, stop.api.stop.lon],
        stop.radius, {
            color: stop.api.route.route_color,
            fillColor: stop.api.route.route_color,
            opacity: stop.backgroundOpacity,
            fillOpacity: stop.backgroundOpacity
        }
    );
};

Bus.prototype.createRow = function(stop) {
    var cellCSS = {
        "color": stop.api.route.route_text_color,
        "background-color": stop.api.route.route_color,
        "opacity": stop.bgOpacity
    };

    // Create our row object
    var row = $("<tr>").css(cellCSS);

    // Create and append the cell containing the route identifier
    // with colored background
    $(row).append($("<td>").text(stop.api.stop.route_id));

    var headsign = $('<span class="headsign">' + stop.api.stop.headsign + '</span>');
    $(row).append($("<td>").append(headsign));

    // Create and append cell with text of departure times
    $(row).append($("<td>").text(stop.departuresText));

    return row;
};

// updateStops runs any manipulation necessary after parsing stops
// into stopList
Bus.prototype.updateStops = function() {
    var self = this;

    // Reset maps
    self.stops = {};
    self.markers = {};
    self.rows = {};

    // Create new table
    var table = $("<table class='table'>");
    var tbody = $("<tbody>");
    var results = $("#results");

    for (var i = 0; i < self.stopList.length; i++) {
        // create the stop row and markers
        var stop = self.stopList[i];
        var row = self.createRow(stop);
        var marker = self.createMarker(stop);

        // Put into maps
        self.stops[stop.id] = stop;
        self.markers[stop.id] = marker;
        self.rows[stop.id] = row;

        console.log(stop, marker, row);

        // Add to display
        $(tbody).append(row);
        marker.addTo(self.map);
    }

    console.log("before", table, tbody, results);

    // Destroy and recreate results
    $(table).append(tbody);
    $(results).empty();
    $(results).append(table);

    console.log("after", table, tbody, results);
};

// getStops calls the stops API with our current state and updates
// the UI with the results
Bus.prototype.getStops = function() {
    var self = this;

    var url = '/api/v2/stops?lat=' + this.lat +
        '&lon=' + this.lon +
        '&filter=' + this.filter +
        '&miles=' + this.miles;

    $.ajax(url, {
        dataType: "json",
        success: function(data) {
            self.parseStops(data);
            self.updateStops();
        },

        error: function(xhr, stat, err) {
            console.log("error in request");
            console.log(xhr, stat, err);
        }
    });
};
