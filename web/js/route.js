// Route is a single instance of a route
function Route(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    self.id = self.api.agency_id + "|" + self.api.route_id;

    self.stop_radius = 10;
    self.stop_fill_color = '#ffffff';

    // markers maps direction id to a map of stop ids mapping to circles. eg:
    // {
    //      0: { "L08N": L.circle(), "L06N": L.circle()},
    //      1: { "L08S": L.circle(), "L06S": L.circle()}
    //  }
    self.markers = self.createMarkers();


    // lines maps direction id to a list of L.polyline objects
    // {
    //      0: [L.poyline()...],
    //      1: [L.polyline()...],
    // }
    self.lines = self.createLines();
}

// createMarkers creates L.circle makers for each stop on the route
Route.prototype.createMarkers = function() {
    var self = this;
    var markers = {}

    for (var dir = 0; dir <= 1; dir++) {
        var dir_markers = {}
        for (var i = 0; i < self.api.stops.length; i++) {
            var stop = self.api.stops[i];
            if (stop.direction_id != dir) {
                continue;
            }

            var circle = L.circle([stop.lat, stop.lon],
                self.stop_radius, {
                    color: self.api.route_color,
                    fillColor: self.stop_fill_color,
                    opacity: 1.0
                }
            );

            dir_markers[stop.stop_id] = circle;
        }

        markers[dir] = dir_markers;
    }

    return markers;
};

// createLines creates L.polyline() objects for each shape of the route 
Route.prototype.createLines = function() {
    var self = this;
    var lines = [];

    // Go through each direction
    for (var dir = 0; dir <= 1; dir++) {
        // Create a list of dir lines
        var dir_lines = [];

        // Go through each route shape
        for (var i = 0; i < self.api.route_shapes.length; i++) {

            // If this shape isn't the current direction, skip it for
            // now
            if (self.api.route_shapes[i].direction_id != dir) {
                continue;
            }

            // Get the shape and init a list of latlons
            var shape = self.api.route_shapes[i];
            var latlons = [];

            // Create a point for each latlon
            for (var j = 0; j < shape.shapes.length; j++) {
                var point = shape.shapes[j];
                latlons[j] = L.latLng(point.lat, point.lon);
            }

            // Create a polyline with the latlons
            var line = L.polyline(
                latlons, {
                    color: self.api.route_color,
                    fillColor: self.api.route_color,
                    opacity: 1.0,
                    fillOpacity: 1.0
                }
            );

            // Append this line to current direction
            dir_lines.push(line);
        };

        // Set this direction's lines
        lines[dir] = dir_lines;
    }

    return lines;
}
