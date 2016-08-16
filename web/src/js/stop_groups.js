function StopGroups(stops) {
    var self = this;

    self.stops = stops;
    self.groups = {};

    self.createGroups();

    console.log(self.groups);
}

StopGroups.prototype.addToGroup = function(stop) {
    var self = this;

    var key = stop.api.agency_id + "|" + stop.api.stop_id + "|" + stop.api.departures[0].compass_dir + "|" + stop.api.group_extra_key;

    if (!self.groups[key]) {
        // If this is the first stop for this group, then create 
        self.groups[key] = {
            route_color: stop.api.route_color,
            route_text_color: stop.api.route_text_color,
            stops: [stop],
        };
    } else {
        // Otherwise just append
        self.groups[key].stops.push(stop);
    }
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
        console.log("yup", self.stops[i]);
        self.addToGroup(self.stops[i]);
    }
};

module.exports = StopGroups;
