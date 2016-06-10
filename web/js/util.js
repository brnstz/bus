// util is a util package
var util = new Util();

function Util() {}

// measure returns the distance in meters between two lat / lon points
// stolen from: 
// http://stackoverflow.com/questions/639695/how-to-convert-latitude-or-longitude-to-meters
Util.prototype.measure = function(lat1, lon1, lat2, lon2) {
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
