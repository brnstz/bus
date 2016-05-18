// Stop is a single instance of a stop
function Stop(api) {
    var self = this;

    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    self.api = api;

    self.map_bg_opacity = 0.2;
    self.map_fg_opacity = 1.0;

    self.table_bg_opacity = 0.5;
    self.table_fg_opacity = 1.0;

    self.radius = 10;

    self.stop_fill_color = '#ffffff';

    // live is true if we have live departures
    self.live = self.isLive();

    // departures is the text of the departures we want to display 
    // in the table
    self.departures = self.createDepartures();
}

// isLive returns true if we have live departures, false if we are using
// scheduled departures
Stop.prototype.isLive = function() {
    var self = this;
    var live = false;

    if (self.api.departures.live != null &&
        self.api.departures.live.length > 0) {
        live = true;
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

    var departures = [];
    var text = "";

    if (self.live) {
        departures = self.api.departures.live;
    } else {
        departures = self.api.departures.scheduled;
    }

    if (departures != null) {
        for (var i = 0; i < departures.length; i++) {
            text += " " + self.timeFormat(departures[i].time);
        }
    }

    return text;
};
