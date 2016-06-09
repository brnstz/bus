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
    self.before_opacity = 0.4;
    self.after_opacity = 1.0;
}

// createMarkers returns a list of L.circle values for this route
// given we are at curstop
Route.prototype.createMarkers = function(curstop) {
    var self = this;
    var markers = [];

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


// createLines returns a list of L.polyline values for this route
// given we are at curstop
Route.prototype.createLines = function(curstop) {
    var self = this;
    var lines = [];


    // Go through each route shape
    for (var i = 0; i < self.api.route_shapes.length; i++) {

        // If this shape is not our direction, then skip it
        if (self.api.route_shapes[i].direction_id != curstop.direction_id) {
            continue;
        }

        // Get the shap in a local var
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
            // transitioning from before to after (FIXME: will these
            // always be exactly the same point?)
            if (before && (point.lat == curstop.lat) && (point.lon = curstop.lon)) {
                before = false;
                // When switching from before to after, always
                // add the last point
                after_latlons.push(L.latLng(point.lat, point.lon));
            }
        }

        // Create a polyline with the latlons
        var before_line = L.polyline(
            before_latlons, {
                color: self.api.route_color,
                fillColor: self.api.route_color,
                opacity: self.before_opacity,
                fillOpacity: self.before_opacity
            }
        );

        // Create a polyline with the latlons
        var after_line = L.polyline(
            after_latlons, {
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
}
