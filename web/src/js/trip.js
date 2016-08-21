var util = require("./util.js");

var stopIcon = L.icon({
    iconUrl: 'img/stop1.svg',
    iconSize: [15, 15]
});

var hereStopIcon = L.icon({
    //iconUrl: 'img/here_stop3.svg',
    iconUrl: 'img/here_red_blink.svg',
    iconSize: [9, 9]
});

// Trip is a single instance of a trip
function Trip(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    // When drawing the lines for the trip, we need to identify when
    // the current stop shows up. Since the line doesn't always pass
    // exactly through the stop, we start by looking for stops
    // at min distance away, then incrementing until we get to max (after
    // which we give up if the line never crosses).
    self.stop_line_dist_min = 0;
    self.stop_line_dist_max = 100;

    self.weight = 8;
    self.before_opacity = 0.5;
    self.after_opacity = 1.0;
}

Trip.prototype.onMap = function(bounds) {
    var self = this;

    if (!self.api.shape_points) {
        return false;
    }

    var shape = self.api.shape_points;

    return util.checkBounds(bounds, shape);
};

// createStopsLabels returns a list of L.circle values for this trip
// given we are at stop
Trip.prototype.createStopsLabels = function(stop) {
    var self = this;
    var stops = [];
    var labels = [];
    var foundStop = false;

    for (var i = 0; i < self.api.stops.length; i++) {
        var tripStop = self.api.stops[i];
        var here = false;
        var marker = null;

        // The first stop gets a bigger radius
        if (tripStop.stop_id == stop.stop_id) {
            foundStop = true;
            here = true;
        }

        // Ignore stops until we find our current stop.
        if (!foundStop) {
            continue;
        }

        marker = L.marker([tripStop.lat, tripStop.lon], {
            icon: stopIcon
        });
        stops.push(marker);
        if (here) {
            stops.push(L.marker([tripStop.lat, tripStop.lon], {
                icon: hereStopIcon
            }));
        }

        var popup = L.popup({
            autoPan: false,
            closeButton: false,
        }, marker);

        popup.setContent(tripStop.stop_name);
        popup.setLatLng([tripStop.lat, tripStop.lon]);
        labels.push(popup);
    }

    return [stops, labels];
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
    for (var d = self.stop_line_dist_min; d <= self.stop_line_dist_max; d++) {

        for (var i = 0; i < self.api.shape_points.length; i++) {
            var point = self.api.shape_points[i];

            if (before) {
                before_latlons.push(L.latLng(point.lat, point.lon));
            } else {
                after_latlons.push(L.latLng(point.lat, point.lon));
            }

            // If the point matches our current stop, then we're
            // transitioning from before to after.
            var difference = util.measure(
                point.lat, point.lon, stop.lat, stop.lon
            );

            if (before && (difference < d)) {
                before = false;
                // When switching from before to after, always
                // add the last point
                after_latlons.push(L.latLng(point.lat, point.lon));
            }
        }

        // If we found the stop, then quit the loop
        if (before == false) {
            break;
        }

        // If this is the final iteration, fall back by using the entire
        // route as the "after" route (full opacity).
        if (before == true && d == self.stop_line_dist_max) {
            after_latlons = before_latlons;
            before_latlons = [];
            break;
        }

        // If we didn't find the stop, then try again with a different d
        // value
        before_latlons = [];
        after_latlons = [];
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
