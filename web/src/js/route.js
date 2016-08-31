var util = require("./util.js");

// Route is a single instance of a route
function Route(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    self.opacity = 1.0;
    self.weight = 1;
    self.routeLines = self.createLines();
};

// createLines returns a list of L.polyline values representing 
// the entire route, without caring about the trip, direction or stops.
Route.prototype.createLines = function() {
    var self = this;
    var lines = [];

    for (var i = 0; i < self.api.route_shapes.length; i++) {
        // Each shape gets its own list of latlons
        var shape = self.api.route_shapes[i];
        var latlons = [];

        // FIXME: temp hack to exclude weird route shapes that appear
        // in LIRR feed
        if (self.api.agency_id == "LI" && self.api.route_long_name != shape.headsign) {
            continue;
        }


        // Create a point for each latlon
        for (var j = 0; j < shape.shapes.length; j++) {
            var point = shape.shapes[j];
            latlons.push(L.latLng(point.lat, point.lon));
        }

        lines.push(L.polyline(
            latlons, {
                weight: self.weight,
                color: self.api.route_color,
                fillColor: self.api.route_color,
                opacity: self.opacity,
                fillOpacity: self.opacity
            }
        ));
    }

    return lines;
};

Route.prototype.onMap = function(bounds) {
    var self = this;

    if (!self.api.route_shapes) {
        return false;
    }

    // Check each shape
    for (var i = 0; i < self.api.route_shapes.length; i++) {
        var shape = self.api.route_shapes[i];

        var found = util.checkBounds(bounds, shape);
        if (found === true) {
            return true;
        }
    }

    return false;
};

module.exports = Route;
