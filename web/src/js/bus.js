// bus is our controller for the bus application. It handles drawing to the
// screen and managing objects.
var bus = new Bus();
var util = require("./util.js");
var Stop = require("./stop.js");
var Route = require("./route.js");
var Trip = require("./trip.js");
var LayerZoom = require("./layer_zoom.js");

var youAreHere = L.icon({
    iconUrl: 'img/here_blue3.svg',
    iconSize: [30, 30]
});

var homeControl = L.Control.extend({
    options: {
        position: 'bottomright'
    },

    onAdd: function(map) {
        return $("<button id='geolocate' type='button' class='btn btn-default' onclick='getbus().geolocate();'><span class='glyphicon glyphicon-screenshot'></span></button>")[0];
    }
});

function Bus() {
    var self = this;

    var nofilter = [];
    var trainsOnly = [0, 1, 2];
    var initialLat = localStorage.getItem("lat");
    var initialLon = localStorage.getItem("lon");

    if (!(initialLat && initialLon)) {
        // default to Times Square
        initialLat = 40.758895;
        initialLon = -73.9873197;
    }

    self.defaultZoom = 16;

    // JSON-encoded Bloom filter (of routes that we have loaded) as 
    // returned by "here" API. Send this back to each "here" request
    // for an update.
    self.filter = '';

    // tileURL is passed to Leaflet JS for drawing the map
    self.tileURL = 'https://stamen-tiles.a.ssl.fastly.net/toner-lite/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    self.tileOptions = {
        maxZoom: 17,
        minZoom: 10,
        opacity: 0.8,
    };

    // mapOptions is the initial options sent on creation of the map
    self.mapOptions = {
        maxZoom: 17,
        minZoom: 10,
        zoom: self.defaultZoom,

        center: [initialLat, initialLon]
    };

    // zoomRouteTypes maps zoom levels to the route types they should send
    self.zoomRouteTypes = {
        10: trainsOnly,
        11: trainsOnly,
        12: trainsOnly,
        13: trainsOnly,
        14: trainsOnly,
        15: nofilter,
        16: nofilter,
        17: nofilter,
    };

    // map is our Leaflet JS map object
    self.map = null;

    // stopList is the list of results in the order returned by the API 
    // (i.e., distance from location)
    self.stopList = [];

    // routes is a mapping from route's unique id to route object
    self.routes = {};

    // rows is stop ids mapped to rows in the results table
    self.rows = {};

    // trip is a mapping from trip's unique id to trip object
    self.trips = {};

    // current_stop is current stop that is clicked
    self.current_stop = null;

    // The current clicked trip
    self.clickedTripLayer = L.featureGroup();

    // Layer of stops on the current clicked trip
    self.stopLayer = L.featureGroup();

    // Layer of stop labels on the current clicked trip
    self.stopLabelLayer = L.featureGroup();

    // Layer of vehicles on the current clicked trip
    self.vehicleLayer = L.featureGroup();

    // Train route shapes
    self.trainRouteLayer = L.featureGroup();

    // Bus route shapes
    self.busRouteLayer = L.featureGroup();

    // layerZooms is a list of LayerZoom objects for each layer on our map.
    // Layers will also be brought to front in order, so "back" layers should
    // go toward the beginning of the list and "front" layers toward the end.
    self.layerZooms = [];

    // true while updating
    self.updating = false;

    // We want to enable the map mover only after our first gelocation
    // request is executed.
    self.firstGeolocate = true;
}

// init is run when the page initially loads
Bus.prototype.init = function() {
    var self = this;

    self.map = L.map('map', self.mapOptions);

    // Add our tiles
    L.tileLayer(self.tileURL, self.tileOptions).addTo(self.map);

    // Create "you are here" marker
    self.marker = L.marker([0, 0], {
        icon: youAreHere
    });
    self.marker.addTo(self.map);

    // Add layers to map
    self.layerZooms.push(new LayerZoom(self.busRouteLayer, 15));
    self.layerZooms.push(new LayerZoom(self.trainRouteLayer, 0));
    self.layerZooms.push(new LayerZoom(self.stopLayer, 13));
    self.layerZooms.push(new LayerZoom(self.stopLabelLayer, 15));
    self.layerZooms.push(new LayerZoom(self.vehicleLayer, 10));
    self.layerZooms.push(new LayerZoom(self.clickedTripLayer, 0));
    self.updateLayers();

    self.map.addControl(new homeControl());

    self.getInitialRoutes();

    self.geolocate();
};

// updateLayers set the visibility and order of layers on each update
Bus.prototype.updateLayers = function() {
    var self = this;

    for (var i = 0; i < self.layerZooms.length; i++) {
        var lz = self.layerZooms[i];
        lz.setVisibility(self.map);
    }
};

Bus.prototype.initMover = function(geoSuccess) {
    var self = this;

    // After the first successful geolocation, set up the move
    // handlers.
    if (self.firstGeolocate) {
        // Set up event handler
        self.map.on("moveend", function() {
            self.getHere();
            self.updateLayers();
        });

        // If we succeeded in doing the geolocate, also set up the watcher
        if (geoSuccess) {

            // Double check
            if (navigator.geolocation) {
                navigator.geolocation.watchPosition(
                    // Success
                    function(p) {
                        self.geoWatchSuccess(p);
                    },

                    // Error (don't need to do anything)
                    null,

                    // Options
                    {
                        enableHighAccuracy: true
                    });
            }
        }

        // Only do this once
        self.firstGeolocate = false;
    }
};

Bus.prototype.geoWatchSuccess = function(p) {
    var self = this;

    // Save last known location
    localStorage.setItem("lat", p.coords.latitude);
    localStorage.setItem("lon", p.coords.longitude);

    // Set location of "you are here" and map view
    self.marker.setLatLng([p.coords.latitude, p.coords.longitude]);
};

Bus.prototype.geoSuccess = function(p) {
    var self = this;

    // Save last known location
    localStorage.setItem("lat", p.coords.latitude);
    localStorage.setItem("lon", p.coords.longitude);

    // Set location of "you are here" and map view
    self.marker.setLatLng([p.coords.latitude, p.coords.longitude]);
    self.map.setView([p.coords.latitude, p.coords.longitude], self.defaultZoom);

    // Remove updating screen
    $("#locating").css("visibility", "hidden");

    // Initialize mover, get results here and update results
    self.initMover(true);
    self.getHere();
    self.updateLayers();
};

Bus.prototype.geoFailure = function() {
    var self = this;

    // The request for location has failed, just get results wherever we were.
    $("#locating").css("visibility", "hidden");

    self.initMover(false);

    self.getHere();
    self.updateLayers();
};

// geolocate requests the location from the browser and sets the location
Bus.prototype.geolocate = function() {
    var self = this;

    if (navigator.geolocation) {
        // Set updating screen
        $("#locating").css("visibility", "visible");

        navigator.geolocation.getCurrentPosition(
            function(p) {
                self.geoSuccess(p);
            },
            function(p) {
                self.geoFailure()
            }, {
                enableHighAccuracy: true
            });

    } else {
        self.geoFailure();
    }
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

    if (data.trips) {
        for (var i = 0; i < data.trips.length; i++) {
            var t = new Trip(data.trips[i]);
            self.trips[t.api.unique_id] = t;
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
        routeID = $("<td class='rowroute'>" + stop.api.route_id + "<br><img src='img/radio1.svg' width=20 height=20></td>");
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

// clickHandler highlights the marker and the row for this stop_id
Bus.prototype.clickHandler = function(stop) {
    var self = this;

    return function(e) {

        if (self.current_stop && self.current_stop.id == stop.id) {
            // If it's the current stop, then just recenter
            self.map.setView([stop.api.lat, stop.api.lon]);
            return;

        } else if (self.current_stop) {
            $(self.rows[self.current_stop.id]).css({
                "opacity": self.current_stop.table_bg_opacity
            });
        }

        var route = self.routes[stop.api.agency_id + "|" + stop.api.route_id];
        var trip = self.trips[stop.api.agency_id + "|" + stop.api.departures[0].trip_id]
        var row = self.rows[stop.id];

        if (!trip) {
            console.log("can't get trip", self.trips, stop.api.agency_id + "|" + stop.api.departures[0].trip_id);
        }

        var sl = trip.createStopsLabels(stop.api);
        var stops = sl[0];
        var labels = sl[1];
        var lines = trip.createLines(stop.api, route.api);
        var vehicles = stop.createVehicles(route.api);
        $(row).css({
            "opacity": stop.table_fg_opacity
        });

        // Clear previous layer elements
        self.clickedTripLayer.clearLayers();
        self.stopLayer.clearLayers();
        self.stopLabelLayer.clearLayers();
        self.vehicleLayer.clearLayers();

        // Add new elements

        // Draw lines 
        for (var i = 0; i < lines.length; i++) {
            self.clickedTripLayer.addLayer(lines[i]);
        }

        // First stop goes on the clicked trip layer (so we always see it)
        if (stops.length > 0) {
            self.clickedTripLayer.addLayer(stops[0]);
        }

        // Draw stops
        for (var i = 1; i < stops.length; i++) {
            self.stopLayer.addLayer(stops[i]);
        }

        // Add stop labels
        for (var i = 0; i < labels.length; i++) {
            //self.stopLabelLayer.addLayer(labels[i]);
        }

        // Draw vehicles
        for (var i = 0; i < vehicles.length; i++) {
            self.vehicleLayer.addLayer(vehicles[i]);
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
        // create the stop row and stops
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
};

Bus.prototype.updateRoutes = function() {
    var self = this;
    var layer = null;

    // Go through each route and add to appropriate layer
    // if it hasn't already been added.
    for (var key in self.routes) {
        var route = self.routes[key];

        if (route.api.route_type_name == "bus") {
            layer = self.busRouteLayer;
        } else {
            layer = self.trainRouteLayer;
        }

        for (var i = 0; i < route.routeLines.length; i++) {
            var line = route.routeLines[i];
            if (!layer.hasLayer(line)) {
                layer.addLayer(line);
            }
        }
    };
};

Bus.prototype.getInitialRoutes = function() {
    var self = this;

    var url = '/api/routes';

    $.ajax(url, {
        dataType: "json",
        success: function(data) {
            self.parseHere(data);
            self.updateRoutes();
        },

        error: function(xhr, stat, err) {
            console.log("error executing routes request");
            console.log(xhr, stat, err);
        }
    });
};

// getHere calls the here API with our current state and updates
// the UI with the results
Bus.prototype.getHere = function() {
    var self = this;

    // Don't update more than once at a time
    if (self.updating) {
        return;
    }

    self.updating = true;

    var center = self.map.getCenter();
    var bounds = self.map.getBounds();
    var sw = bounds.getSouthWest();
    var ne = bounds.getNorthEast();
    var routeTypes = self.zoomRouteTypes[self.map.getZoom()];

    var url = '/api/here' +
        '?lat=' + encodeURIComponent(center.lat) +
        '&lon=' + encodeURIComponent(center.lng) +
        '&sw_lat=' + encodeURIComponent(sw.lat) +
        '&sw_lon=' + encodeURIComponent(sw.lng) +
        '&ne_lat=' + encodeURIComponent(ne.lat) +
        '&ne_lon=' + encodeURIComponent(ne.lng) +
        '&filter=' + encodeURIComponent(self.filter);

    for (var i = 0; i < routeTypes.length; i++) {
        url += '&route_type=' + encodeURIComponent(routeTypes[i]);
    }

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

// initbus should be called by the windows to initialize the bus object
window.initbus = function() {
    bus.init();
};

// getbus allows you to retrieve the core bus object in the console for
// debugging
window.getbus = function() {
    return bus;
};
