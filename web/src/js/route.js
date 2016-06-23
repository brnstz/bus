var util = require("./util.js");
var Bezier = require("bezier-js");

// Route is a single instance of a route
function Route(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    self.id = self.api.agency_id + "|" + self.api.route_id;

    self.stop_radius = 15;
    self.outline_color = '#000000';

    // before/after opacity is the opacity of stops before/after
    // us in the stop sequence
    self.before_opacity = 0.2;
    self.after_opacity = 1.0;

    self.weight = 4;

    // stop_line_dist is the number of meters we assume
    // a stop can be from the polyline to say that it hit the stop
    self.stop_line_dist = 10;
};

// createMarkers returns a list of L.circle values for this route
// given we are at curstop
Route.prototype.createMarkers = function(curstop) {
    var self = this;
    var markers = [];

    if (!self.api.stops) {
        return markers;
    }

    for (var i = 0; i < self.api.stops.length; i++) {
        var stop = self.api.stops[i];

        // Ignore stops that aren't going our direction
        if (stop.direction_id != curstop.direction_id) {
            continue;
        }

        // by default, fill with white
        var fill_color = '#ffffff'
        if (stop.stop_id == curstop.stop_id) {
            fill_color = '#000000';
        }


        // ignore stops before our stop
        if (stop.stop_sequence < curstop.stop_sequence) {
            continue;
        }

        var circle = L.circle([stop.lat, stop.lon],
            self.stop_radius, {
                width: 1,
                color: self.api.route_color,
                fillColor: fill_color,
                opacity: self.after_opacity,
                fillOpacity: self.after_opacity
            }
        );

        markers.push(circle);
    }

    return markers;
};

Route.prototype.createGlobalLines = function(curstop, overlap) {
    console.log("what am I?", overlap);
    var self = this;
    var lines = [];

    if (!self.api.route_shapes) {
        return lines;
    }

    // Go through each route shape
    for (var i = 0; i < self.api.route_shapes.length; i++) {
        var latlons = [];

        // If this shape is not our direction, then skip it
        if (self.api.route_shapes[i].direction_id != curstop.direction_id) {
            continue;
        }

        // Get the shape in a local var
        var shape = self.api.route_shapes[i];

        // Create a point for each latlon
        // Set up p1 for initial iteration
        var p1 = shape.shapes[0];
        var last_offset = 0;
        var offset_points = [p1];

        for (var j = 1; j < shape.shapes.length; j++) {
            var p2 = shape.shapes[j];
            var offset = overlap.add(p1.lat, p1.lon, p2.lat, p2.lon);
            console.log("offset is", offset);

            if (offset == last_offset) {
                console.log("matching offset", offset, last_offset);
                // While offset is the same, push to list
                offset_points.push(p2);

            } else {
                console.log("mismatching offset", offset, last_offset);
                // Copy and reinit the list
                var now_points = offset_points;
                offset_points = [p2];

                if (last_offset == 0) {
                    // If it's zero, nothing special to do, just add
                    // the points.
                    for (var k = 0; k < now_points.length; k++) {
                        var np = now_points[k];
                        latlons.push(L.latLng(np.lat, np.lon));
                    }
                } else {
                    // Otherwise, we need to offset them, so convert to 
                    // format that bezier likes
                    var coords = [];
                    for (var k = 0; k < now_points.length; k++) {
                        var np = now_points[k];
                        coords.push(np.lat);
                        coords.push(np.lon);
                    }

                    // Create bezier obj and create points with offset
                    var bezier = new Bezier(coords);
                    var bzpoints = line.offset(offset);

                    // Add all points to latlons (offset returns a list
                    // of objects, each of which has points).
                    for (var l = 0; l < bzpoints.length; l++) {
                        for (var m = 0; m < bzpoints[l].points.length; m++) {
                            var p = bzpoints[l].points[m];
                            latlons.push(L.latLng(p.x, p.y));
                        }
                    }

                    // Set up for next iteration
                    p1 = p2;
                }

                // Create a polyline with the latlons
                var line = L.polyline(
                    latlons, {
                        weight: self.weight,
                        color: self.api.route_color,
                        fillColor: self.api.route_color,
                        opacity: self.after_opacity,
                        fillOpacity: self.after_opacity
                    }
                );

                lines.push(line);
            };
        };

        return lines;
    }
};


// createLines returns a list of L.polyline values for this route
// given we are at curstop
Route.prototype.createLines = function(curstop) {
    var self = this;
    var lines = [];

    if (!self.api.route_shapes) {
        return lines;
    }

    // Go through each route shape
    for (var i = 0; i < self.api.route_shapes.length; i++) {

        // If this shape is not our direction, then skip it
        if (self.api.route_shapes[i].direction_id != curstop.direction_id) {
            continue;
        }

        // Get the shape in a local var
        var shape = self.api.route_shapes[i];

        // Create a list of before and after latlons (different drawing 
        // style before and after our stop)
        var before_latlons = [];
        var after_latlons = [];

        // Assume we're before our stop until hearing otherwise
        var before = true;

        // Create a point for each latlon
        for (var j = 0; j < shape.shapes.length; j++) {
            var point = shape.shapes[j];

            if (before) {
                before_latlons.push(L.latLng(point.lat, point.lon));
            } else {
                after_latlons.push(L.latLng(point.lat, point.lon));
            }

            // If the point matches our current stop, then we're
            // transitioning from before to after.
            var difference = util.measure(point.lat, point.lon, curstop.lat, curstop.lon);
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
                color: self.api.route_color,
                fillColor: self.api.route_color,
                opacity: self.before_opacity,
                fillOpacity: self.before_opacity
            }
        );

        // Create a polyline with the latlons
        var after_line = L.polyline(
            after_latlons, {
                weight: self.weight,
                color: self.api.route_color,
                fillColor: self.api.route_color,
                opacity: self.after_opacity,
                fillOpacity: self.after_opacity
            }
        );

        lines.push(before_line);
        lines.push(after_line);

    };

    return lines;
};

Route.prototype.createVehicles = function(curstop) {
    var self = this;
    var vehicles = [];

    if (!curstop.vehicles) {
        return vehicles;
    }

    for (var i = 0; i < curstop.vehicles.length; i++) {
        var v = curstop.vehicles[i];
        /* FIXME
        var opts = {
            color: self.api.route_color
        };
        var bounds = [
            [v.lat, v.lon],
            [v.lat + .000001, v.lon + 000001]
        ];
        var square = L.rectangle(bounds, opts);
        vehicles.push(square);
        */

        var opts = {
            color: '#000000'
        };

        var black = L.circleMarker([v.lat, v.lon], opts);
        vehicles.push(black);

    }

    return vehicles;
};

module.exports = Route;
