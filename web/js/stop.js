// Stop is a single instance of a stop
function Stop(api) {
    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    this.api = api;

    this.bg_opacity = 0.5;
    this.fg_opacity = 1.0;
    this.radius = 20;

    // live is true if we have live departures
    this.live = this.isLive();

    // departures is the text of the departures we want to display 
    // in the table
    this.departures = this.createDepartures();
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
