// bus is our controller for the bus application. It handles drawing to the
// screen and managing objects.
var StopGroups = require("./stop_groups.js");
var util = require("./util.js");
var Stop = require("./stop.js");
var Route = require("./route.js");
var Trip = require("./trip.js");
var LayerZoom = require("./layer_zoom.js");
var isMobile = require("ismobilejs");
var bus = new Bus();

var youAreHere = L.icon({
    iconUrl: 'img/here_blue3.svg',
    iconSize: [30, 30]
});

var homeControl = L.Control.extend({
    options: {
        position: 'bottomright'
    },

    onAdd: function(map) {
        return $("<button id='geolocate' type='button' class='btn btn-default' onclick='getbus().geolocate();'><img src='img/gps_solid.svg' height='20' width='20'></button>")[0];
    }

});

function Bus() {
    var self = this;

    // When at a close zoom level, we don't use a route type filter
    var nofilter = [];

    // At a wider level, we only want trains (0-2) and ferrys (4)
    var highfilter = [0, 1, 2, 4];

    // See if we stored our last location in local storage
    var initialLat = localStorage.getItem("lat");
    var initialLon = localStorage.getItem("lon");

    // default location when there are no results or we never visited
    // before
    self.timesSquare = {
        lat: 40.758895,
        lon: -73.9873197,
    };

    // If we didn't, default to Times Square
    if (!(initialLat && initialLon)) {
        initialLat = self.timesSquare.lat;
        initialLon = self.timesSquare.lon;
    }

    self.defaultZoom = 16;
    self.maxZoom = 17;
    self.minZoom = 8;

    // JSON-encoded Bloom filter (of routes that we have loaded) as returned by
    // "here" API. Send this back to each "here" request for an update.
    self.filter = '';

    // tileURL is passed to Leaflet JS for drawing the map
    self.tileURL = 'https://stamen-tiles.a.ssl.fastly.net/toner-lite/{z}/{x}/{y}.png';

    // tileOptions is passed to Leatlef JS for drawing the map
    self.tileOptions = {
        maxZoom: self.maxZoom,
        minZoom: self.minZoom,
        opacity: 1.0,
    };

    // mapOptions is the initial options sent on creation of the map
    self.mapOptions = {
        maxZoom: self.maxZoom,
        minZoom: self.minZoom,
        zoom: self.defaultZoom,

        center: [initialLat, initialLon]
    };

    // zoomRouteTypes maps zoom levels to the route types they should send
    self.zoomRouteTypes = {
        10: highfilter,
        11: highfilter,
        12: highfilter,
        13: highfilter,
        14: highfilter,
        15: nofilter,
        16: nofilter,
        17: nofilter,
    };

    // map is our Leaflet JS map object
    self.map = null;

    // FIXME: we should use only topgroups
    // stopList is the list of results in the order returned by the
    // API (i.e., distance from location)
    //self.stopList = [];

    // stop groups is a mapping of unique ids that groups
    // stops together (e.g., NQR going nw)
    self.stopGroups = new StopGroups([]);

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

    // layerZooms is a list of LayerZoom objects for each layer on our
    // map. Layers will also be brought to front in order, so "back"
    // layers should go toward the beginning of the list and "front"
    // layers toward the end.
    self.layerZooms = [];

    // true while updating
    self.updating = false;

    // We want to enable the map mover only after our first gelocation
    // request is executed.
    self.firstGeolocate = true;

    // The current inflight "here" req if any.
    self.here_req = null;

    // Increment the request id so we don't display results for
    // oudated requests.
    self.here_req_id = 0;
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
                        enableHighAccuracy: isMobile.any
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
    $("#loading").css("visibility", "hidden");

    // Initialize mover, get results here and update results
    self.initMover(true);

    self.getHere();
};

Bus.prototype.geoFailure = function() {
    var self = this;

    // The request for location has failed, just get results wherever we were.
    $("#loading").css("visibility", "hidden");

    self.initMover(false);

    self.getHere();
};

// geolocate requests the location from the browser and sets the location
Bus.prototype.geolocate = function() {
    var self = this;

    if (navigator.geolocation) {
        // Set updating screen
        $("#loading").css("visibility", "visible");

        navigator.geolocation.getCurrentPosition(
            function(p) {
                self.geoSuccess(p);
            },
            function(p) {
                self.geoFailure()
            }, {
                enableHighAccuracy: isMobile.any
            });

    } else {
        self.geoFailure();
    }
};

// parseHere reads the text of response from the here API and updates
// stops and routes
Bus.prototype.parseHere = function(data) {
    var self = this;
    var stoplist = [];

    if (data.stops) {

        // Create a stop object for each result and save to our list
        for (var i = 0; i < data.stops.length; i++) {
            var s = new Stop(data.stops[i]);
            stoplist[i] = s;
        }

        self.stopGroups = new StopGroups(stoplist);

    } else {
        self.stopGroups = new StopGroups([]);

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

Bus.prototype.createGroupRow = function(sg) {
    var self = this;
    var now = new Date();
    var mins = parseInt((sg.min_departure - now) / 1000 / 60)

    var cellCSS = {
        "color": sg.route_text_color,
        "background-color": sg.route_color,
    }

    var row = $("<tr>");
    $(row).css(cellCSS);

    var td1 = $("<td class='sgdir'>" + "<img src='img/compass_plain.svg' style='transform: rotate(" + sg.compass_dir + "deg);' width=20 height=20></td>");
    var td2 = $("<td class='sgroutes'>" +
        "<span class='routenames'>" + sg.display_names + "</span>" +
        "<br>" +
        "<span class='stopname'>" + sg.stop_name + "</span>" +
        "</td>");
    //var td3 = $("<td class='rowroute'>" + sg.stop_name + "</td>");
    var td4 = $("<td class='sgmin'>" + mins + " min</td>");

    $(row).append(td1);
    $(row).append(td2);
    //$(row).append(td3);
    $(row).append(td4);

    return row;
};

// createRow creates a results row for this stop
Bus.prototype.createRow = function(stop, i) {
    var self = this;

    var route = self.routes[stop.api.agency_id + "|" + stop.api.route_id];
    var opacity;

    if (self.current_stop && stop.api.unique_id == self.current_stop.api.unique_id) {
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
    /*
    if (stop.live == true) {
        routeID = $("<td class='rowroute'>" + stop.api.route_id + "<br><img src='img/radio1.svg' width=20 height=20></td>");
    } else {
        routeID = $("<td class='rowroute'>" + stop.api.route_id + "</td>");
    }
    */


    routeID = $("<td class='rowroute'>" + stop.api.route_id + "<br><img src='img/compass_plain.svg' style='transform: rotate(" + stop.api.departures[0].compass_dir + "deg);' width=20 height=20></td>");


    var datatd = $("<td>");
    var headsign = $('<span class="headsign">' + stop.api.trip_headsign + '</span>');
    var departures = $('<span><br>' + stop.departures + '</span>');
    $(row).append(routeID);
    $(datatd).append(headsign);
    $(datatd).append(departures);
    $(row).append(datatd);

    return row;
};

// createEmptyRow creates a single empty row indicating there are
// no stops on the map
Bus.prototype.createEmptyRow = function() {
    var self = this;

    var cellCSS = {
        "color": "#222222",
        "background-color": "#ffffff",
        "opacity": 1.0
    };

    var row = $("<tr>");
    var td = $("<td>");
    var a = $("<a href='#'>Times Square</a>").click(function() {
        self.map.setView([self.timesSquare.lat, self.timesSquare.lon], self.defaultZoom);
        self.getHere();
        return false;
    });

    $(td).append("No departures in this area. Try ");
    $(td).append(a);
    $(td).append("?");
    $(row).css(cellCSS);
    $(row).append(td);

    return row;
};

// getRoute returns a promise to get a route when it was a false positive in
// the bloom filter
Bus.prototype.getRoute = function(agency_id, route_id) {
    var self = this;

    var url = '/api/route' +
        '?agency_id=' + encodeURIComponent(agency_id) +
        '&route_id=' + encodeURIComponent(route_id);

    var promise = $.ajax(url, {
        dataType: "json"
    });

    promise.fail(function(xhr, text_status, error) {
        console.log("failed", xhr, text_status, error);
    });

    promise.done(function(data) {
        var r = new Route(data);
        self.routes[r.api.unique_id] = r;
    });

    return promise;
};

// getTrip returns a promise to get a trip when it was a false 
// positive in the bloom filter
Bus.prototype.getTrip = function(agency_id, route_id, trip_id) {
    var self = this;

    var url = '/api/trip' +
        '?agency_id=' + encodeURIComponent(agency_id) +
        '&route_id=' + encodeURIComponent(route_id) +
        '&trip_id=' + encodeURIComponent(trip_id);

    var promise = $.ajax(url, {
        dataType: "json"
    });

    promise.fail(function(xhr, text_status, error) {
        console.log("failed", xhr, text_status, error);
    });

    promise.done(function(data) {
        var t = new Trip(data);
        self.trips[t.api.unique_id] = t;
    });

    return promise;
};



// clickHandler highlights the marker and the row for this stop_id
Bus.prototype.clickHandler = function(stop) {
    var self = this;

    return function(e) {

        if (self.current_stop && self.current_stop.api.unique_id == stop.api.unique_id) {
            // If it's the current stop, then just recenter
            self.map.setView([stop.api.lat, stop.api.lon]);
            return;

        } else if (self.current_stop) {
            $(self.rows[self.current_stop.api.unique_id]).css({
                "opacity": self.current_stop.table_bg_opacity
            });
        }

        var route_promise;
        var trip_promise;

        if (!self.routes[stop.api.agency_id + "|" + stop.api.route_id]) {
            console.log("getting route via promise", stop.api.agency_id, stop.api.route_id);
            route_promise = self.getRoute(stop.api.agency_id, stop.api.route_id);
        } else {
            route_promise = $("<div>").promise();
        }

        if (!self.trips[stop.api.agency_id + "|" + stop.api.departures[0].trip_id]) {
            console.log("getting trip via promise", stop.api.agency_id, stop.api.route_id, stop.api.departures[0].trip_id);

            trip_promise = self.getTrip(stop.api.agency_id, stop.api.route_id, stop.api.departures[0].trip_id);
        } else {
            trip_promise = $("<div>").promise();
        }

        route_promise.done(function() {
            trip_promise.done(function() {


                var route = self.routes[stop.api.agency_id + "|" + stop.api.route_id];
                var trip = self.trips[stop.api.agency_id + "|" + stop.api.departures[0].trip_id]
                var row = self.rows[stop.api.unique_id];
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

                // First stop goes on the clicked trip layer (so we always see
                // it)
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
            });
        });

    };
}

// updateStops runs any manipulation necessary after parsing stops
// into stopList
Bus.prototype.updateStops = function() {
    var self = this;
    var stop = null;

    // Reset rows
    self.rows = {};

    // Ensure that current stop still represents a route that
    // is on screen
    if (self.current_stop != null) {
        stop = self.current_stop;
        var route = self.routes[stop.api.agency_id + "|" + stop.api.route_id];
        var trip = self.trips[stop.api.agency_id + "|" + stop.api.departures[0].trip_id]
        var bounds = self.map.getBounds();

        if (!(route.onMap(bounds) || trip.onMap(bounds))) {
            self.current_stop = null;
        }
    }

    // Create new table
    var table = $("<table class='results'>");
    var tbody = $("<tbody>");
    var results = $("#results");

    for (var i = 0; i < self.stopGroups.keys.length; i++) {
        var key = self.stopGroups.keys[i];
        var sg = self.stopGroups.groups[key];
        var row = self.createGroupRow(sg);
        $(tbody).append(row);
    }

    /* 
     // FIXME: This is how we used to draw rows
    // If there's a current stop, show it first
    if (self.current_stop != null) {
        self.stopList.unshift(self.current_stop);
    }

    for (var i = 0; i < self.stopList.length; i++) {
        // create the stop row and stops
        stop = self.stopList[i];

        // If the current stop shows up after first, then ignore it
        if (i != 0 && self.current_stop && self.current_stop.api.unique_id == stop.api.unique_id) {
            continue;
        }

        var row = self.createRow(stop, i);

        // Put into row
        self.rows[stop.api.unique_id] = row;

        // Add to row display
        $(tbody).append(row);

        var handler = self.clickHandler(stop);
        $(row).click(handler);
    }
    */

    // Set first result to current stop if none selected
    /* FIXME: this is how we used to click the first result
    if (self.current_stop == null && self.stopList.length > 0) {
        var row = self.rows[self.stopList[0].api.unique_id];
        $(row).trigger("click");
    }

    if (self.stopList.length === 0) {
        $(tbody).append(self.createEmptyRow());
    }
    */

    // Destroy and recreate results
    $(table).append(tbody);
    $(results).empty();
    $(results).append(table);
    $(results).animate({
        "scrollTop": 0
    }, "fast");
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
            self.updateLayers();
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

    $("#loading").css("visibility", "visible");

    // Abort any previous requests in flight
    if (self.here_req != null) {
        self.here_req.abort();
    }

    // Update the here id
    self.here_req_id++;
    var here_req_now = self.here_req_id;

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

    self.here_req = $.ajax(url, {
        dataType: "json",

        success: function(data) {
            if (self.here_req_id == here_req_now) {
                // If our request id is the most recent one, then
                // process the response and reset the request to null
                self.parseHere(data);
                self.updateStops();
                self.updateRoutes();
                self.updateLayers();

                self.here_req = null;
                $("#loading").css("visibility", "hidden");
            }

            // Otherwise, we ignore the response because we have 
            // something more recent in flight
        },

        error: function(xhr, stat, err) {
            if (self.here_req_id == here_req_now) {

                // If our request id is the most recent one, then 
                // process the response

                // Usually this will be an abort request, but if it's
                // not then log the error
                if (err != "abort") {
                    console.log("error executing here request");
                    console.log(xhr, stat, err);
                    $("#loading").css("visibility", "hidden");
                }

                // Reset this to null though typically when this is the
                // result of abort, the primary request will immediately
                // reset this. But this seems to be the right thing to do.
                self.here_req = null;
            }

            // Otherwise, we ignore the response because we have 
            // something more recent in flight
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
