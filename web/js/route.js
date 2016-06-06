// Route is a single instance of a route
function Route(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    self.id = self.api.agency_id + "|" + self.api.route_id;

    self.stop_radius = 10;
    self.stop_fill_color = '#ffffff';

    // markers maps stop_ids to L.circle() objects representing the stop
    self.markers = self.createMarkers();

    // lines is a list of L.polyline() objects representing the full path
    // of the route
    self.lines = self.createLines();
}

// createMarkers creates L.circle makers for each stop on the route
Route.prototype.createMarkers = function() {
    var self = this;
    var markers = {};

    for (var i = 0; i < self.api.stops.length; i++) {
        var stop = self.api.stops[i];
        var circle = L.circle([stop.lat, stop.lon],
            self.stop_radius, {
                color: self.api.route_color,
                fillColor: self.stop_fill_color,
                opacity: 1.0
            }
        );

        markers[stop.stop_id] = circle;
    }

    return markers;
};

// createLines creates L.polyline() objects for each shape of the route 
Route.prototype.createLines = function() {
    var self = this;
    var lines = [];

    for (var i = 0; i < self.api.route_shapes.length; i++) {
        var shape = self.api.route_shapes[i];
        var latlons = [];

        for (var j = 0; j < shape.shapes.length; j++) {
            var point = shape.shapes[j];
            latlons[j] = L.latLng(point.lat, point.lon);
        }

        var line = L.polyline(
            latlons, {
                color: self.api.route_color,
                fillColor: self.api.route_color,
                opacity: 1.0,
                fillOpacity: 1.0
            }
        );

        lines[i] = line;
    };

    return lines;
}
