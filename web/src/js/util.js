// measure returns the distance in meters between two lat / lon points
// stolen from: 
// http://stackoverflow.com/questions/639695/how-to-convert-latitude-or-longitude-to-meters

module.exports.measure = function(lat1, lon1, lat2, lon2) {
    var R = 6378.137;
    var dLat = (lat2 - lat1) * Math.PI / 180;
    var dLon = (lon2 - lon1) * Math.PI / 180;
    var a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
        Math.cos(lat1 * Math.PI / 180) * Math.cos(lat2 * Math.PI / 180) *
        Math.sin(dLon / 2) * Math.sin(dLon / 2);
    var c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    var d = R * c;

    return d * 1000;
}

// checkBounds checks that a shape is within the bounds
module.exports.checkBounds = function(bounds, shape) {
    var last_point = null;
    var last_ll = null;

    // Check each point
    for (var j = 0; j < shape.length; j++) {

        var ll = L.latLng(shape[j].lat, shape[j].lon);
        var point = L.point(shape[j].lat, shape[j].lon);

        if (last_point == null && bounds.contains(ll) === true) {
            return true;

        } else if (last_point != null) {
            var line = L.bounds([last_point, point]);
            var other_bounds = L.latLngBounds(last_ll, ll);

            if (bounds.intersects(other_bounds)) {
                return true;
            }
        }

        last_point = point;
        last_ll = ll;
    }

    return false;
}
