function Result(result) {
    // result is the object returned by the API. We add next to this
    // things that are relevant to JS only. 
    this.result = result;

    this.backgroundOpacity = 0.50;
    this.foregroundOpacity = 1.0;

    // live is true if we have live departures
    this.live = this.isLive();

    // departuresText is the text of the departures we want to display 
    // in the table
    this.departuresText = this.createDeparturesText();

    // marker is the marker we should draw on the map
    this.marker = this.createMarker();

    // row is the row in the result table
    this.row = this.createRow();
}

// createMarker builds the map marker for this stop
Result.prototype.createMarker = function() {
    var opt = {
        color: this.result.route.route_color,
        fillColor: this.result.route.route_color,
        opacity: this.backgroundOpacity,
        fillOpacity: this.backgroundOpacity
    };
    var radius = 10;
    var latlon = [this.result.stop.lat, this.result.stop.lon];

    return L.circle(latlon, radius, opt);
};

// isLive returns true if we have live departures, false if we are using
// scheduled departures
Result.prototype.isLive = function() {
    var self = this;
    var live = false;

    if (self.result.departures.live != null &&
        self.result.departures.live.length > 0) {
        live = true;
    }

    return live;
};

// timeFormat takes a time object and returns the format we want to display on
// screen
Result.prototype.timeFormat = function(time) {
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


Result.prototype.createDeparturesText = function() {
    var self = this;

    var departures = [];
    var text = "";

    if (self.live) {
        departures = self.result.departures.live;
    } else {
        departures = self.result.departures.scheduled;
    }

    console.log(self);

    if (departures != null) {
        for (var i = 0; i < departures.length; i++) {
            text += " " + self.timeFormat(departures[i].time);
        }
    }

    return text;
};

Result.prototype.createRow = function() {
    var self = this;

    // Create our empty row in the background
    var row = document.createElement("tr");
    row.style.opacity = this.backgroundOpacity;

    // Create cell for the name of the route
    var routeCell = document.createElement("td");
    var routeCellText = document.createTextNode(self.result.stop.route_id);

    // Set the route cell's color
    routeCell.style.color = self.result.route.route_text_color;
    routeCell.style.backgroundColor = self.result.route.route_color;

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
        " " + self.result.stop.headsign + " " + self.departuresText
    );

    // Append span and text to the cell, then to the row
    infoCell.appendChild(infoSpan);
    infoCell.appendChild(infoText);
    row.appendChild(infoCell);

    return row;
};

// foreground puts this result in the foreground
Result.prototype.foreground = function() {
    this.marker.setStyle({
        opacity: this.foregroundOpacity,
        fillOpacity: this.foregroundOpacity
    });

    this.marker.bringToFront();

    this.row.style.opacity = this.foregroundOpacity;
};

// background puts this result in the background
Result.prototype.background = function() {
    this.marker.setStyle({
        opacity: this.backgroundOpacity,
        fillOpacity: this.backgroundOpacity
    });

    this.row.style.opacity = this.backgroundOpacity;
};
