function Result(result) {
    // result is the object returned by the API. We add next to this
    // things that are relevant to JS only. 
    this.result = result;

    // marker is the marker we should draw on the map
    this.marker = this.createMarker();
}

// createMarker builds the map marker for this stop
Result.prototype.createMarker = function() {
    var opt = {
        color: "#" + this.result.route.route_color,
        fillColor: "#" + this.result.route.route_color,
        opacity: 0.2,
        fillOpacity: 0.2
    };
    var radius = 10;
    var latlon = [this.result.stop.lat, this.result.stop.lon];

    return L.circle(latlon, radius, opt);
};
