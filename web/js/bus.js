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

    // here is our marker for current location
    this.here = null;

    // stopList is the list of results in the order returned by the API 
    // (i.e., distance from location)
    this.stopList = [];

    // stops is stop ids mapped to stop objects
    this.stops = {};

    // markers is stop ids mapped to markers on the map
    this.markers = {};

    // paths is route_ids+direction_ids mapped to L.polyline paths on the map 
    this.paths = {};

    // rows is stop ids mapped to rows in the results table
    this.rows = {};
}

// init is run when the page initially loads
Bus.prototype.init = function() {
    var self = this;

    self.map = L.map('map');
    self.marker = L.marker([0, 0]);

    self.map.on("dragend", function() {
        var ll = self.map.getCenter();
        self.updatePosition(ll.lat, ll.lng);
    });

    self.marker.addTo(self.map);

    self.geolocate();
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
    for (var i = 0; i < data.results.length; i++) {
        (function() {
            var s = new Stop(data.results[i]);
            self.stopList[i] = s;
        }());
    }

    self.updateStops();
};

// createMarker creates the map marker for this stop
Bus.prototype.createMarker = function(stop) {
    var self = this;

    return L.circle(
        [stop.api.stop.lat, stop.api.stop.lon],
        stop.radius, {
            color: stop.path_color,
            fillColor: stop.stop_fill_color,
            opacity: stop.map_fg_opacity,
            fillOpacity: stop.map_fg_opacity
        }
    );
};

// createRow creates a results row for this stop
Bus.prototype.createRow = function(stop, i) {
    var cellCSS = {
        "color": stop.api.route.route_text_color,
        "background-color": stop.api.route.route_color,
        "opacity": stop.table_bg_opacity
    };

    // Create our row object
    var row = $("<tr>");
    $(row).css(cellCSS);

    // Create and append the cell containing the route identifier
    // with colored background
    $(row).append($("<td>").text(stop.api.route.route_id))

    var headsign = $('<span class="headsign">' + stop.api.stop.headsign + '</span>');
    $(row).append($("<td>").append(headsign));

    // Create and append cell with text of departure times
    $(row).append($("<td>").text(stop.departures));

    return row;
};

// createPath draws the path of this stop's display trip
Bus.prototype.createPath = function(stop) {
    var self = this;
    var latlons = [];

    // If there is a cached version, return that
    var cached = self.paths[stop.api.stop.route_id + stop.api.stop.direction_id];

    // If there is no response, then return null
    if (!(stop.api.display_trip && stop.api.display_trip.shape_points)) {
        return null;
    }

    // Otherwise go through each point and create path
    for (var i = 0; i < stop.api.display_trip.shape_points.length; i++) {
        var point = stop.api.display_trip.shape_points[i];
        latlons[i] = L.latLng(point.lat, point.lon);
    }

    var line = L.polyline(
        latlons, {
            color: stop.path_color,
            fillColor: stop.api.route.route_color,
            opacity: stop.map_fg_opacity,
            fillOpacity: stop.map_fg_opacity
        }
    );

    return line;
}

// clickHandler highlights the marker and the row for this stop_id
Bus.prototype.clickHandler = function(stop_id) {
    var self = this;

    return function(e) {
        for (var i = 0; i < self.stopList.length; i++) {
            var stop = self.stopList[i];
            var marker = self.markers[stop.api.id];
            var row = self.rows[stop.api.id];
            var path = self.paths[stop.api.stop.route_id + stop.api.stop.direction_id];

            if (stop.api.id == stop_id) {
                if (path !== null) {
                    path.setStyle({
                        color: stop.api.route.route_color,
                        fillColor: stop.api.route.route_color,
                        opacity: stop.map_fg_opacity,
                        fillOpacity: stop.map_fg_opacity
                    });

                    path.bringToFront();
                }


                // If it's the current stop, set fg opacity and bring to
                // front
                $(row).css("opacity", stop.table_fg_opacity);
                marker.setStyle({
                    color: stop.api.route.route_color,
                    fillOpacity: stop.map_fg_opacity,
                    opacity: stop.map_fg_opacity
                });
                marker.bringToFront();


                self.map.setView([stop.api.stop.lat, stop.api.stop.lon]);


            } else {
                // All other stops, set to bg opacity and bring to 
                // background
                $(row).css("opacity", stop.table_bg_opacity);
                marker.setStyle({
                    color: stop.path_color,
                    fillOpacity: stop.map_bg_opacity,
                    opacity: stop.map_bg_opacity
                });
                marker.bringToBack();

                if (path !== null) {
                    path.setStyle({
                        color: stop.path_color,
                        opacity: stop.map_bg_opacity,
                        fillOpacity: stop.map_bg_opacity
                    });

                    path.bringToBack();
                }
            }
        }
    };
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
        var row = self.createRow(stop, i);
        var marker = self.createMarker(stop);
        var path = self.createPath(stop);

        // Put into maps
        self.stops[stop.api.id] = stop;
        self.markers[stop.api.id] = marker;
        self.rows[stop.api.id] = row;
        self.paths[stop.api.stop.route_id + stop.api.stop.direction_id] = path;

        // Add to display
        $(tbody).append(row);
        marker.addTo(self.map);
        if (path !== null) {
            path.addTo(self.map);
        }

        var handler = self.clickHandler(stop.api.id);
        marker.on('click', handler);
        $(row).click(handler);
    }

    // Destroy and recreate results
    $(table).append(tbody);
    $(results).empty();
    $(results).append(table);
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
        },

        error: function(xhr, stat, err) {
            console.log("error in request");
            console.log(xhr, stat, err);
        }
    });
};
