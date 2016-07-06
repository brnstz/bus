// LayerZoom assocates layers with the minZoom level they should be visible at.
function LayerZoom(layer, minZoom) {
    var self = this;

    self.layer = layer;
    self.minZoom = minZoom;
}

LayerZoom.prototype.setVisibility = function(map) {
    var self = this;

    // Add/remove bus layer at the appropriate zoom levels
    if (map.getZoom() >= self.minZoom) {
        if (!map.hasLayer(self.layer)) {
            map.addLayer(self.layer);
        }
        self.layer.bringToFront();

    } else {
        if (map.hasLayer(self.layer)) {
            map.removeLayer(self.layer);
        }
    }
};

module.exports = LayerZoom;
