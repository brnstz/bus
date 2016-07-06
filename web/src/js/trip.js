var util = require("./util.js");

var stopIcon = L.icon({
    iconUrl: 'img/stop1.svg',
    //iconSize: [30, 30]
    iconSize: [15, 15]
});

// Trip is a single instance of a trip
function Trip(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    // stop_line_dist is the number of meters we assume
    // a stop can be from the polyline to say that it hit the stop
    self.stop_line_dist = 50;

    self.stop_radius = 20;

    self.first_stop_radius = 20;

    self.weight = 10;
    self.before_opacity = 0.5;
    self.after_opacity = 1.0;
}

// createMarkers returns a list of L.circle values for this trip
// given we are at stop
Trip.prototype.createMarkers = function(stop, route) {
    var self = this;
    var markers = [];
    var foundStop = false;

    for (var i = 0; i < self.api.stops.length; i++) {
        var tripStop = self.api.stops[i];
        var radius = 0;

        // The first stop gets a bigger radius
        if (tripStop.stop_id == stop.stop_id) {
            radius = self.first_stop_radius;
            foundStop = true;
        } else {
            radius = self.stop_radius;
        }

        // Ignore stops until we find our current stop.
        if (!foundStop) {
            continue;
        }

        markers.push(L.marker([tripStop.lat, tripStop.lon], {
            icon: stopIcon
        }));
    }

    return markers;
};

// createLines returns a list of L.polyline values for this trip
// given we are at curstop
Trip.prototype.createLines = function(stop, route) {
    var self = this;
    var lines = [];

    // Create a list of before and after latlons (different drawing 
    // style before and after our stop)
    var before_latlons = [];
    var after_latlons = [];

    // Assume we're before our stop until hearing otherwise
    var before = true;

    // Create a point for each latlon
    for (var i = 0; i < self.api.shape_points.length; i++) {
        var point = self.api.shape_points[i];

        if (before) {
            before_latlons.push(L.latLng(point.lat, point.lon));
        } else {
            after_latlons.push(L.latLng(point.lat, point.lon));
        }

        // If the point matches our current stop, then we're
        // transitioning from before to after.
        var difference = util.measure(point.lat, point.lon, stop.lat, stop.lon);
        if (before && (difference < self.stop_line_dist)) {

            before = false;
            // When switching from before to after, always
            // add the last point
            after_latlons.push(L.latLng(point.lat, point.lon));
        }
    }

    // Create a polyline with the latlons
    var before_line = L.polyline(
        before_latlons, {
            weight: self.weight,
            color: route.route_color,
            fillColor: route.route_color,
            opacity: self.before_opacity,
            fillOpacity: self.before_opacity
        }
    );

    // Create a polyline with the latlons
    var after_line = L.polyline(
        after_latlons, {
            weight: self.weight,
            color: route.route_color,
            fillColor: route.route_color,
            opacity: self.after_opacity,
            fillOpacity: self.after_opacity
        }
    );

    lines.push(before_line);
    lines.push(after_line);

    return lines;
};

module.exports = Trip;
