function Result(result) {
    // result is the object returned by the API. We add next to this
    // things that are relevant to JS only. 
    this.result = result;

    this.backgroundOpacity = 0.2;
    this.foregrounOpacity = 1.0;

    // marker is the marker we should draw on the map
    this.marker = this.createMarker();
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

// foreground puts this result in the foreground
Result.prototype.foreground = function() {
    this.marker.setStyle({
        opacity: this.foregroundOpacity,
        fillOpacity: this.foregroundOpacity
    });

    this.marker.bringToFront();
};

// background puts this result in the background
Result.prototype.background = function() {
    this.marker.setStyle({
        opacity: this.backgroundOpacity,
        fillOpacity: this.backgroundOpacity
    });
};
