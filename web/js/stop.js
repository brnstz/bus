// Stop is a single instance of a stop
function Stop(api) {
    // api is the object returned by the API. We leave this as read-only
    // and add any other info we want as a sibling data piece.
    this.api = api;

    this.bgOpacity = 0.50;
    this.fgOpacity = 1.0;
    this.radius = 10;

    // live is true if we have live departures
    this.live = this.isLive();

    // departuresText is the text of the departures we want to display 
    // in the table
    this.departuresText = this.createDeparturesText();
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


Stop.prototype.createDeparturesText = function() {
    var self = this;

    var departures = [];
    var text = "";

    if (self.live) {
        departures = self.api.departures.live;
    } else {
        departures = self.api.departures.scheduled;
    }

    console.log(self);

    if (departures != null) {
        for (var i = 0; i < departures.length; i++) {
            text += " " + self.timeFormat(departures[i].time);
        }
    }

    return text;
};

Stop.prototype.createRow = function() {
    var self = this;

    // Create our empty row in the background
    var row = document.createElement("tr");
    row.style.opacity = self.bgOpacity;

    // Create cell for the name of the route
    var routeCell = document.createElement("td");
    var routeCellText = document.createTextNode(self.api.stop.route_id);

    // Set the route cell's color
    routeCell.style.color = self.api.route.route_text_color;
    routeCell.style.backgroundColor = self.api.route.route_color;

    // Add route cell to the row
    routeCell.appendChild(routeCellText);
    row.appendChild(routeCell);

    // Create info cell
    var infoCell = document.createElement("td");

    // infoSpan is a span for a glyphicon arrow
    var infoSpan = document.createElement("span");
    infoSpan.classList.add("glyphicon");
    infoSpan.classList.add("glyphicon-arrow-right");
    infoSpan.setAttribute("aria-hidden", "true");

    // infoText is text of the direction and departures
    var infoText = document.createTextNode(
        " " + self.api.stop.headsign + " " + self.departuresText
    );

    // Append span and text to the cell, then to the row
    infoCell.appendChild(infoSpan);
    infoCell.appendChild(infoText);
    row.appendChild(infoCell);

    return row;
};

// foreground puts this result in the foreground
Stop.prototype.foreground = function() {
    var self = this;

    self.marker.setStyle({
        opacity: self.fgOpacity,
        fillOpacity: self.fgOpacity
    });

    self.marker.bringToFront();

    self.row.style.opacity = self.fgOpacity;
};

// background puts this result in the background
Stop.prototype.background = function() {
    var self = this;

    self.marker.setStyle({
        opacity: self.bgOpacity,
        fillOpacity: self.bgOpacity
    });

    self.row.style.opacity = self.bgOpacity;
};
