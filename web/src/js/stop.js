var trainIcon = L.icon({
    iconUrl: 'img/train1_circle.svg',
    iconSize: [30, 30]
});

var busIcon = L.icon({
    iconUrl: 'img/bus1_circle.svg',
    iconSize: [30, 30]
});

function getIcon(route_type_name) {
    var icon = null;

    switch (route_type_name) {
        case "train", "subway":
            icon = trainIcon;
            break;

        case "bus":
            icon = busIcon;
            break;

        default:
            icon = trainIcon;
    }

    return icon;
}

// Stop is a single instance of a stop
function Stop(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    self.id = self.api.agency_id + "|" + self.api.route_id + "|" + self.api.stop_id;

    self.map_fg_opacity = 1.0;

    self.table_bg_opacity = 0.3;
    self.table_fg_opacity = 1.0;

    // live is true if we have live departures
    self.live = self.isLive();

    // departures is the text of the departures we want to display 
    // in the table
    self.departures = self.createDepartures();
}

// isLive returns true if we have live departures, false if we are using
// only scheduled departures
Stop.prototype.isLive = function() {
    var self = this;
    var live = false;

    var departures = self.api.departures;

    if (departures != null) {
        for (var i = 0; i < departures.length; i++) {
            if (departures[i].live == true) {
                live = true;
                break;
            }
        }
    }

    return live;
};

// timeFormat takes a time object and returns the format we want to display on
// screen
Stop.prototype.timeFormat = function(time) {
    var self = this;

    var t = new Date(time);

    // Get minutes as a 00 padded value
    var minutes = ("00" + t.getMinutes()).slice(-2);

    // Get a temporary value for hours, format it below
    var hours = t.getHours();

    // Convert to US time
    if (hours > 12) {
        hours -= 12;
    }
    if (hours == 0) {
        hours = 12;
    }

    return hours + ":" + minutes;
}


Stop.prototype.createDepartures = function() {
    var self = this;

    var text = "";

    var departures = self.api.departures;

    if (departures != null) {
        for (var i = 0; i < departures.length; i++) {
            text += " " + self.timeFormat(departures[i].time);
        }
    }

    return text;
};

Stop.prototype.createVehicles = function(route) {
    var self = this;
    var vehicles = [];

    for (var i = 0; i < self.api.vehicles.length; i++) {
        var v = self.api.vehicles[i];
        var icon;

        var opts = {
            icon: getIcon(route.route_type_name)
        };

        vehicles.push(L.marker([v.lat, v.lon], opts));
    }

    return vehicles;
};

module.exports = Stop;
