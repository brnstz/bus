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
    //
    // FIXME: these should all be local vars / parameters
    self.lat = 0;
    self.lon = 0;
    self.sw_lat = 0;
    self.sw_lon = 0;
    self.ne_lat = 0;
    self.ne_lon = 0;

    // JSON-encoded Bloom filter (of routes that we have loaded) as 
    // returned by "here" API. Send this back to each "here" request
    // for an update.
    self.filter = '';

    // tileURL is passed to Leaflet JS for drawing the map
    self.tileURL = 'https://stamen-tiles.a.ssl.fastly.net/toner-lite/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    self.tileOptions = {
        maxZoom: 18,
        minZoom: 5,
        opacity: 0.8,
    };

    // mapOptions is the initial options sent on creation of the map
    self.mapOptions = {
        maxZoom: 18,
        minZoom: 5,
        zoom: 16,

        // default to Times Square
        center: [40.758895, -73.9873197]
    };

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

    // The current clicked trip
    self.clickedTripLayer = L.layerGroup();

    // Train route shapes
    self.trainRouteLayer = L.layerGroup();

    // Bus route shapes
    self.busRouteLayer = L.layerGroup();

    // The zoom level at which busRouteLayer is visible
    self.minBusZoom = 10;

    // true while updating
    self.updating = false;

    // Avoid weird iPhone bouncing: http://stackoverflow.com/a/26853900
    //self.firstMove = false;
}

// init is run when the page initially loads
Bus.prototype.init = function() {
    var self = this;

    // Avoid weird iPhone bouncing: http://stackoverflow.com/a/26853900
    /*
    window.addEventListener('touchstart', function(e) {
        self.firstMove = true;
    });
    window.addEventListener('touchmove', function(e) {
        if (self.firstMove) {
            e.preventDefault();

            self.firstMove = false;
        }
    });
    */

    self.map = L.map('map', self.mapOptions);

    // Add our tiles
    L.tileLayer(self.tileURL, self.tileOptions).addTo(self.map);

    // Create "you are here" marker
    self.marker = L.marker([0, 0]);

    // Set up event handler
    /* FIXME: would this be good enough?
    self.map.on("moveend", self.movend);
    self.map.on("zoomend", self.zoomend);
    */
    // Set up event handlers
    self.map.on("moveend", function() {
        self.moveend();
    });
    self.map.on("zoomend", function() {
        self.zoomend();
    });

    self.marker.addTo(self.map);

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

// zoomend ensures the correct layers are shown on the map after
// a zoom event
Bus.prototype.zoomend = function() {
    var self = this;

    // Add/remove bus layer at the appropriate zoom levels
    if (self.map.getZoom() >= self.minBusZoom) {
        if (!self.map.hasLayer(self.busRouteLayer)) {
            self.map.addLayer(self.busRouteLayer);
        }
    } else {
        if (self.map.hasLayer(self.busRouteLayer)) {
            self.map.removeLayer(self.busRouteLayer);
        }
    }
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
                p.coords.longitude
            );
        });
    }
};

// refresh re-requests stops from the current position
Bus.prototype.refresh = function() {
    var self = this;

    // Get the results for this location
    self.getHere();
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
    var sw = bounds.getSouthWest();
    var ne = bounds.getNorthEast();
    self.sw_lat = sw.lat;
    self.sw_lon = sw.lng;
    self.ne_lat = ne.lat;
    self.ne_lon = ne.lng;

    // Get the results for this location
    self.getHere();
};

// parseHere reads the text of response from the here API and updates
// stops and routes
Bus.prototype.parseHere = function(data) {
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

    if (data.routes) {
        for (var i = 0; i < data.routes.length; i++) {
            var r = new Route(data.routes[i]);
            self.routes[r.api.unique_id] = r;
        };
    }

    if (data.filter) {
        self.filter = JSON.stringify(data.filter);
    }
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
    var routeID = null;
    if (stop.live == true) {
        routeID = $("<td class='rowroute'>" + stop.api.route_id + "<br><img src='img/radio.png' width=20 height=20></td>");
    } else {
        routeID = $("<td class='rowroute'>" + stop.api.route_id + "</td>");
    }


    var datatd = $("<td>");
    var headsign = $('<span class="headsign">' + stop.api.headsign + '</span>');
    var departures = $('<span><br>' + stop.departures + '</span>');
    $(row).append(routeID);
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
        var vehicles = route.createVehicles(stop.api);
        $(row).css({
            "opacity": stop.table_fg_opacity
        });

        // First clear the map of any existing routes
        self.clear();

        var vals = [];

        // Draw lines 
        for (var i = 0; i < lines.length; i++) {
            vals.push(lines[i]);
        }

        // Draw marker stops
        for (var key in markers) {
            vals.push(markers[key]);
        }

        // Draw vehicles
        for (var key in vehicles) {
            vals.push(vehicles[key]);
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

        /* FIXME: ignore this for now
        var handler = self.clickHandler(stop);
        $(row).click(handler);
        */
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

Bus.prototype.updateRoutes = function() {
    var self = this;
    var layer = null;

    // Go through each route and add to appropriate layer
    // if it hasn't already been added.
    for (var key in self.routes) {
        var route = self.routes[key];

        if (route.api.route_type == "bus") {
            layer = self.busRouteLayer;
        } else {
            layer = self.trainRouteLayer;
        }

        if (!layer.hasLayer(route.api.routeLines)) {
            layer.addLayer(route.api.routeLines);
        }
    };

};

// getHere calls the here API with our current state and updates
// the UI with the results
Bus.prototype.getHere = function() {
    var self = this;

    var url = '/api/here' +
        '?lat=' + encodeURIComponent(self.lat) +
        '&lon=' + encodeURIComponent(self.lon) +
        '&sw_lat=' + encodeURIComponent(self.sw_lat) +
        '&sw_lon=' + encodeURIComponent(self.sw_lon) +
        '&ne_lat=' + encodeURIComponent(self.ne_lat) +
        '&ne_lon=' + encodeURIComponent(self.ne_lon) +
        '&filter=' + encodeURIComponent(self.filter);

    $.ajax(url, {
        dataType: "json",
        success: function(data) {
            self.parseHere(data);
            self.updateStops();
            self.updateRoutes();
            self.updating = false;
        },

        error: function(xhr, stat, err) {
            console.log("error executing here request");
            console.log(xhr, stat, err);
            self.updating = false;
        }
    });
};

window.initbus = function() {
    bus.init();
};
