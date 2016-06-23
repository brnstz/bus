// Overlap tells us if a line overlaps an already existing line
function Overlap() {
    var self = this;

    // a map of "x,y,x,y" values to integers. eg
    // "40.721938,-73.9537543,40.7142157,-73.9516788" => 1
    console.log("Hello there");
    self.overlap = {};
}

// add the lat lon to our overlap list and return how many lines
// this line overlaps with.
Overlap.prototype.add = function(x1, y1, x2, y2) {
    var self = this;

    var fwd = [x1, y1, x2, y2].join(",");
    var rev = [x2, y2, x1, y1].join(",");

    var count = 0;

    // If we have it in fwd direction, increment local count to return
    // and value in cache
    if (self.overlap[fwd]) {
        count += self.overlap[fwd];
        self.overlap[fwd] += 1;
    } else {
        // Otherwise set first overlap in fwd direction
        self.overlap[fwd] = 1;
    }

    // Also count reverse direction
    if (self.overlap[rev]) {
        count += self.overlap[rev];
    }

    return count;
}

module.exports = Overlap;
