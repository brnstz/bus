// bus is our controller for the bus application. It handles drawing to the
// screen and managing objects.
var bus = new Bus();

function Bus() {
    var self = this;

    // lat, lon is the center of our request. We send this to the Bus API
    // and also use it to draw the map. We can get this value from the
    // HTML5 location API.
    self.lat = 0;
    self.lon = 0;

    // miles and filter are options sent to the Bus API
    self.miles = 0.5;
    self.filter = '';

    // tileURL is passed to Leaflet JS for drawing the map
    self.tileURL = 'https://stamen-tiles.a.ssl.fastly.net/toner-lite/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    self.tileOptions = {
        MaxZoom: 20
    };

    // zoom is the initial zoom value when drawing the Leaflet map
    self.zoom = 16;

    // map is our Leaflet JS map object
    self.map = null;

    // here is our marker for current location
    self.here = null;

    // stopList is the list of results in the order returned by the API 
    // (i.e., distance from location)
    self.stopList = [];

    // routes is a mapping from route_id to route object
    self.routes = {};

    // rows is stop ids mapped to rows in the results table
    self.rows = {};

    // current_stop is current stop that is clicked
    self.current_stop = null;

    // layer is the current layer on the map
    self.layer = null;

    // true while updating
    self.updating = false;
}

// init is run when the page initially loads
Bus.prototype.init = function() {
    var self = this;

    self.map = L.map('map');
    self.marker = L.marker([0, 0]);

    self.map.on("dragend", function() {
        self.dragend();
    });

    self.marker.addTo(self.map);

    self.geolocate();
};

Bus.prototype.dragend = function() {
    var self = this;

    // Only process one update at a time.
    if (self.updating) {
        return;
    }

    // This must be done with self.updating = false somewhere
    // after self.updatePosition is called
    self.updating = true;

    var ll = self.map.getCenter();
    self.updatePosition(ll.lat, ll.lng);
};

// refresh requests the location from the browser, sets our lat / lon and
// gets new trips from the API 
Bus.prototype.geolocate = function() {
    var self = this;

    if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(function(p) {
            self.marker.setLatLng([p.coords.latitude, p.coords.longitude]);
            self.updatePosition(
                p.coords.latitude,
                p.coords.longitude,
                self.zoom
            );
        });
    }
};

Bus.prototype.refresh = function() {
    var self = this;

    // Get the results for this location
    self.getStops();
};

// updatePosition takes an HTML5 geolocation position response and 
// updates our map and trip info
Bus.prototype.updatePosition = function(lat, lon, zoom) {
    var self = this;

    // Set our lat and lon based on the coords
    self.lat = lat;
    self.lon = lon;

    // Set location and zoom of the map.
    self.map.setView([self.lat, self.lon], zoom);

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
    for (var i = 0; i < data.stops.length; i++) {
        var s = new Stop(data.stops[i]);
        self.stopList[i] = s;
    }

    // After we parseStops we need to get any missing routes
    self.getRoutes();
};

Bus.prototype.parseRoutes = function(data) {
    var self = this;

    for (var i = 0; i < data.routes.length; i++) {
        var r = new Route(data.routes[i]);
        self.routes[r.id] = r;
    };

    // After we parseRoutes, it's time to updateStops
    self.updateStops();
};

// createRow creates a results row for this stop
Bus.prototype.createRow = function(stop, i) {
    var self = this;

    var route = self.routes[stop.api.agency_id + "|" + stop.api.route_id];
    if (!route) {
        console.log(stop.api.agency_id + "|" + stop.api.route_id);
    }

    var cellCSS = {
        "color": route.api.route_text_color,
        "background-color": route.api.route_color,
        "opacity": stop.table_bg_opacity
    };

    // Create our row object
    var row = $("<tr>");
    $(row).css(cellCSS);

    // Create and append the cell containing the route identifier
    // with colored background
    $(row).append($("<td>").text(stop.api.route_id))

    var headsign = $('<span class="headsign">' + stop.api.headsign + '</span>');
    $(row).append($("<td>").append(headsign));

    // Create and append cell with text of departure times
    $(row).append($("<td>").text(stop.departures));

    return row;
};

// clear removes the current route from map
Bus.prototype.clear = function() {
    var self = this;

    // Nothing to do
    if (self.layer == null || self.map == null) {
        return
    }

    self.layer.clearLayers();
};

// clickHandler highlights the marker and the row for this stop_id
Bus.prototype.clickHandler = function(stop) {
    var self = this;

    return function(e) {

        // If it's the current stop, then just recenter
        if (self.current_stop && self.current_stop.id == stop.id) {
            self.map.setView([stop.api.lat, stop.api.lon]);
            return;
        }

        var route = self.routes[stop.api.agency_id + "|" + stop.api.route_id];
        var row = self.rows[stop.api.id];
        var markers = route.createMarkers(stop.api);
        var lines = route.createLines(stop.api);

        // First clear the map of any existing routes
        self.clear();

        // Then recenter and draw
        self.map.setView([stop.api.lat, stop.api.lon]);

        var vals = [];

        // Draw lines 
        for (var i = 0; i < lines.length; i++) {
            vals.push(lines[i]);
        }

        // Draw marker stops
        for (var key in markers) {
            vals.push(markers[key]);
        }

        self.layer = L.layerGroup(vals);
        self.layer.addTo(self.map);

        self.current_stop = stop;
    };
}

// updateStops runs any manipulation necessary after parsing stops
// into stopList
Bus.prototype.updateStops = function() {
    var self = this;

    // Reset rows
    self.rows = {};

    // Create new table
    var table = $("<table class='table'>");
    var tbody = $("<tbody>");
    var results = $("#results");

    for (var i = 0; i < self.stopList.length; i++) {
        // create the stop row and markers
        var stop = self.stopList[i];
        var row = self.createRow(stop, i);

        // Put into row
        self.rows[stop.id] = row;

        // Add to row display
        $(tbody).append(row);

        var handler = self.clickHandler(stop);
        $(row).click(handler);
    }

    // Destroy and recreate results
    $(table).append(tbody);
    $(results).empty();
    $(results).append(table);

    self.updating = false;
};

// getStops calls the stops API with our current state and updates
// the UI with the results
Bus.prototype.getStops = function() {
    var self = this;

    var url = '/api/stops' +
        '?lat=' + encodeURIComponent(this.lat) +
        '&lon=' + encodeURIComponent(this.lon) +
        '&filter=' + encodeURIComponent(this.filter) +
        '&miles=' + encodeURIComponent(this.miles);

    $.ajax(url, {
        dataType: "json",
        success: function(data) {
            self.parseStops(data);
        },

        error: function(xhr, stat, err) {
            console.log("error in request");
            console.log(xhr, stat, err);
            self.updateStops();
        }
    });
};

// getRoutes calls the routes API for any routes 
// we don't yet have
Bus.prototype.getRoutes = function() {
    var self = this;
    var params = [];
    var uniq = {};

    // Iterate through stops in the response and get info on
    // any routes we don't have
    for (var i = 0; i < self.stopList.length; i++) {
        var stop = self.stopList[i];
        var uniq_id = stop.api.agency_id + "|" + stop.api.route_id;

        // Ignore routes already found in this call
        if (uniq[uniq_id]) {
            continue;
        }

        // Ignore routes we already pulled
        if (self.routes[uniq_id]) {
            continue;
        }

        params.push('agency_id=' + encodeURIComponent(stop.api.agency_id));
        params.push('route_id=' + encodeURIComponent(stop.api.route_id));
    }

    if (params.length > 0) {
        // Only make the call when there are any routes to add
        var url = '/api/routes?' + params.join("&");
        $.ajax(url, {
            dataType: "json",
            success: function(data) {
                self.parseRoutes(data);
            },

            error: function(xhr, stat, err) {
                console.log("error in request");
                console.log(xhr, stat, err);
                self.updateStops();
            }
        });
    } else {
        // Otherwise we just need to update the stops
        self.updateStops();
    }
};
