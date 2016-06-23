// bus is our controller for the bus application. It handles drawing to the
// screen and managing objects.
var bus = new Bus();
var util = require("./util.js");
var Stop = require("./stop.js");
var Route = require("./route.js");
var Bezier = require("bezier-js");

var homeControl = L.Control.extend({
    options: {
        position: 'bottomright'
    },

    onAdd: function(map) {
        return $("<button id='geolocate' type='button' class='btn btn-primary' onclick='bus.geolocate();'><span class='glyphicon glyphicon-screenshot'></span></button>")[0];
    }
});

var refreshControl = L.Control.extend({
    options: {
        position: 'bottomright'
    },

    onAdd: function(map) {
        return $("<button id='refresh' type='button' class='btn btn-success' onclick='bus.refresh();'><span class='glyphicon glyphicon-refresh'></span></button>")[0];
    }
});

function Bus() {
    var self = this;

    // lat, lon is the center of our request. We send this to the Bus API
    // and also use it to draw the map. We can get this value from the
    // HTML5 location API.
    self.lat = 0;
    self.lon = 0;

    // miles and filter are options sent to the Bus API
    self.filter = '';

    // tileURL is passed to Leaflet JS for drawing the map
    self.tileURL = 'https://stamen-tiles.a.ssl.fastly.net/toner-lite/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    self.tileOptions = {
        MaxZoom: 20
    };

    // zoom is the initial zoom value when drawing the Leaflet map
    self.zoom = 16;

    self.meters = 1000;

    // map is our Leaflet JS map object
    self.map = null;

    // stopList is the list of results in the order returned by the API 
    // (i.e., distance from location)
    self.stopList = [];

    // routes is a mapping from route_id to route object
    self.routes = {};

    // rows is stop ids mapped to rows in the results table
    self.rows = {};

    // current_stop is current stop that is clicked
    self.current_stop = null;

    // last_stop is the stop that was clicked second most recently
    self.last_stop = null;

    // layer is the current highlighted layer on the map
    self.layer = L.layerGroup();

    // global_layer the accumulated global layer
    self.global_layer = L.layerGroup();

    // true while updating
    self.updating = false;
}

// init is run when the page initially loads
Bus.prototype.init = function() {
    var self = this;

    self.map = L.map('map');

    // Add our tiles
    L.tileLayer(self.tileURL, self.tileOptions).addTo(self.map);

    // Create "you are here" marker
    self.marker = L.marker([0, 0]);

    self.map.on("moveend", function() {
        self.moveend();
    });

    self.marker.addTo(self.map);

    self.layer.addTo(self.map);
    self.global_layer.addTo(self.map);

    self.map.addControl(new homeControl());
    self.map.addControl(new refreshControl());

    self.geolocate();
};

// movend gets the current center of the maps and gets new data 
// based on the location
Bus.prototype.moveend = function() {
    var self = this;

    var ll = self.map.getCenter();
    self.updatePosition(ll.lat, ll.lng);
};

// geolocate requests the location from the browser, sets our lat / lon and
// gets new trips from the API 
Bus.prototype.geolocate = function() {
    var self = this;

    if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(function(p) {
            // Set location of "you are here"
            self.marker.setLatLng([p.coords.latitude, p.coords.longitude]);

            // update our position with current geolocation
            self.updatePosition(
                p.coords.latitude,
                p.coords.longitude,
                self.zoom
            );
        });
    }
};

// refresh re-requests stops from the current position
Bus.prototype.refresh = function() {
    var self = this;

    // Get the results for this location
    self.getStops();
};

// updatePosition takes an HTML5 geolocation position response and 
// updates our map and trip info
Bus.prototype.updatePosition = function(lat, lon, zoom) {
    var self = this;

    // Don't update more than once at a time
    if (self.updating) {
        return;
    }

    // This is set to false in self.updateStops()
    self.updating = true;

    // Set our lat and lon based on the coords
    self.lat = lat;
    self.lon = lon;

    // Set location and zoom of the map.
    self.map.setView([self.lat, self.lon], zoom);

    var bounds = self.map.getBounds();
    var nw = bounds.getNorthWest();
    var distance = util.measure(self.lat, self.lon, nw.lat, nw.lng);

    self.meters = distance;

    // Get the results for this location
    self.getStops();
};

// parseStops reads the text of response from the stops API and updates
// the initial list of stop objects
Bus.prototype.parseStops = function(data) {
    var self = this;

    if (data.stops) {
        // Reset list of stops
        self.stopList = [];

        // Create a stop object for each result and save to our list
        for (var i = 0; i < data.stops.length; i++) {
            var s = new Stop(data.stops[i]);
            self.stopList[i] = s;
        }
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

    var opacity = 0;
    if (self.current_stop && stop.id == self.current_stop.id) {
        opacity = stop.table_fg_opacity;
    } else {
        opacity = stop.table_bg_opacity;
    }

    var cellCSS = {
        "color": route.api.route_text_color,
        "background-color": route.api.route_color,
        "opacity": opacity
    };

    // Create our row object
    var row = $("<tr>");
    $(row).css(cellCSS);

    // Create and append the cell containing the route identifier
    // with colored background
    $(row).append($("<td class='rowroute'>").text(stop.api.route_id))

    var datatd = $("<td>");
    var headsign = $('<span class="headsign">' + stop.api.headsign + '</span>');
    var departures = $('<span><br>' + stop.departures + '</span>');
    $(datatd).append(headsign);
    $(datatd).append(departures);
    $(row).append(datatd);


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
        } else if (self.current_stop) {
            $(self.rows[self.current_stop.id]).css({
                "opacity": self.current_stop.table_bg_opacity
            });
        }

        var route = self.routes[stop.api.agency_id + "|" + stop.api.route_id];
        var row = self.rows[stop.id];
        var markers = route.createMarkers(stop.api);
        var lines = route.createLines(stop.api);
        var lines2 = route.createLines(stop.api);
        var vehicles = route.createVehicles(stop.api);
        $(row).css({
            "opacity": stop.table_fg_opacity
        });

        // First clear the map of any existing routes
        self.clear();

        var vals = [];

        // Draw lines 
        for (var i = 0; i < lines.length; i++) {
            self.layer.addLayer(lines[i]);
            self.global_layer.addLayer(lines2[i]);
        }

        // Draw marker stops
        for (var key in markers) {
            self.layer.addLayer(markers[key]);
        }

        // Draw vehicles
        for (var key in vehicles) {
            self.layer.addLayer(vehicles[key]);
        }


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

    // If there's a current stop, show it first
    if (self.current_stop != null) {
        self.stopList.unshift(self.current_stop);
    }

    for (var i = 0; i < self.stopList.length; i++) {
        // create the stop row and markers
        var stop = self.stopList[i];

        // If the current stop shows up after first, then ignore it
        if (i != 0 && self.current_stop && self.current_stop.id == stop.id) {
            continue;
        }

        var row = self.createRow(stop, i);

        // Put into row
        self.rows[stop.id] = row;

        // Add to row display
        $(tbody).append(row);

        var handler = self.clickHandler(stop);
        $(row).click(handler);
    }

    // Set first result to current stop if none selected
    if (self.current_stop == null && self.stopList.length > 0) {
        var row = self.rows[self.stopList[0].id];
        $(row).trigger("click");
    }

    // Destroy and recreate results
    $(table).append(tbody);
    $(results).empty();
    $(results).append(table);

    self.updating = false;

    // If we rejected a move, our position might be off. Trigger
    // another update.
    var ll = self.map.getCenter();
    if (!(self.lat == ll.lat && self.lon == ll.lng)) {
        self.updatePosition(ll.lat, ll.lng);
    }
};

// getStops calls the stops API with our current state and updates
// the UI with the results
Bus.prototype.getStops = function() {
    var self = this;

    var url = '/api/stops' +
        '?lat=' + encodeURIComponent(self.lat) +
        '&lon=' + encodeURIComponent(self.lon) +
        '&filter=' + encodeURIComponent(self.filter) +
        '&meters=' + encodeURIComponent(self.meters);

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

window.initbus = function() {
    bus.init();
};
