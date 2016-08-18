function StopGroups(stops) {
    var self = this;

    // allow for 8 possible directions
    self.roundfactor = 360 / 8;

    self.stops = stops;
    self.groups = {};
    self.keys = [];

    self.createGroups();
}

StopGroups.prototype.addToGroup = function(stop) {
    var self = this;

    var roundedCompass = Math.round(stop.api.departures[0].compass_dir / self.roundfactor) * self.roundfactor;

    var key = stop.api.agency_id + "|" + stop.api.stop_id + "|" + roundedCompass + "|" + stop.api.group_extra_key;

    if (!self.groups[key]) {
        // If this is the first stop for this group, then create 
        self.groups[key] = {
            route_color: stop.api.route_color,
            route_text_color: stop.api.route_text_color,
            stops: [stop],
            compass_dir: roundedCompass,
            key: key
        };

        // Add it to ordered list of keys
        self.keys.push(key);

    } else {
        // Otherwise just append
        self.groups[key].stops.push(stop);
    }
};

StopGroups.prototype.init = function(sg) {
    var self = this;
    var display_names = {};
    var min_departure = null;
    var now = new Date();

    for (var i = 0; i < sg.stops.length; i++) {
        var stop = sg.stops[i];

        // Record all route display names
        display_names[stop.api.display_name] = true;

        // Get the first departure
        var t = new Date(stop.api.departures[0].time);
        if (min_departure == null) {
            min_departure = t;
        } else if (t < min_departure) {
            min_departure = t;
        }
    };

    sg.display_names = Object.keys(display_names).join(", ");
    sg.stop_name = sg.stops[0].api.stop_name;
    sg.min_departure = min_departure;

    sg.stops = sg.stops.sort(function(a, b) {
        var da = new Date(a.api.departures[0].time);
        var db = new Date(b.api.departures[0].time);

        return da > db;
    });
};

StopGroups.prototype.createGroups = function() {
    var self = this;

    // mapping of:
    // "agency_id|stop_id|compass_dir|extra_group_key" => {
    //      stops: [list of stops],
    //      etc..
    //  }
    // 
    for (var i = 0; i < self.stops.length; i++) {
        self.addToGroup(self.stops[i]);
    }

    // Put any finishing touches on the final groups
    for (var i = 0; i < self.keys.length; i++) {
        var key = self.keys[i];
        var sg = self.groups[key];

        self.init(sg);
    }
};

module.exports = StopGroups;
