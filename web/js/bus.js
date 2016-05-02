var bus = new Bus();

function Bus() {
    this.lat = 0;
    this.lon = 0;
    this.miles = 0.5;
    this.filter = '';
}

// refresh requests the location from the browser, sets our lat / lon and
// gets new trips from the API 
Bus.prototype.refresh = function() {
    var self = this;
    if (navigator.geolocation) {
        navigator.geolocation.getCurrentPosition(function(position) {
            self.lat = position.coords.latitude;
            self.lon = position.coords.longitude;
            self.getTrips();
        });
    }
};

// appendCell creates a td cell with the value and appends it to row
Bus.prototype.appendCell = function(row, value) {
    var cell = document.createElement("td");
    var cellText = document.createTextNode(value);

    cell.appendChild(cellText);
    row.appendChild(cell);
};

// appendCell creates a td cell with the value and appends it to row
// along with setting the fg and bg colors
Bus.prototype.appendCellColor = function(row, value, fgcolor, bgcolor) {
    var cell = document.createElement("td");
    var cellText = document.createTextNode(value);

    cell.style.color = "#" + fgcolor;
    cell.style.backgroundColor = "#" + bgcolor;

    cell.appendChild(cellText);
    row.appendChild(cell);
};

// appendTime adds a cell to this row with the current time values
Bus.prototype.appendTime = function(row, times) {
    var mytext = "";

    if (times != null) {
        for (var i = 0; i < times.length; i++) {
            var mytime = new Date(times[i].time);
            var h = mytime.getHours();
            var ampm = "am";
            if (h >= 12) {
                h = h - 12;
                ampm = "pm";
            } else if (h == 0) {
                h = 12;
            }
            var hour = ("00" + h).slice(-2);
            var minute = ("00" + mytime.getMinutes()).slice(-2);
            mytext = mytext + " " + hour + ":" + minute + ampm;
        }
    }

    this.appendCell(row, mytext);
};

// getTrips calls the stops API with our current state and updates
// the UI with the results
Bus.prototype.getTrips = function() {
    var self = this;
    var xhr = new XMLHttpRequest();
    var url = '/api/v2/stops?lat=' + this.lat +
        '&lon=' + this.lon +
        '&filter=' + this.filter +
        '&miles=' + this.miles;

    xhr.open('GET', url);
    xhr.onload = function(e) {
        var data = JSON.parse(this.response);
        var tbl = document.createElement("table");
        tbl.setAttribute("class", "table");
        var tblBody = document.createElement("tbody");
        var results = document.getElementById("results");

        if (results.childNodes.length > 0) {
            results.removeChild(results.childNodes[0]);
        }

        for (var i = 0; i < data.results.length; i++) {
            var res = data.results[i];
            var row = document.createElement("tr");

            self.appendCellColor(row, res.stop.route_id, res.route.route_text_color, res.route.route_color);
            self.appendCell(row, res.stop.stop_name);
            self.appendCell(row, res.stop.headsign);

            if (res.departures.live != null && res.departures.live.length > 0) {
                self.appendTime(row, res.departures.live);
            } else {
                self.appendTime(row, res.departures.scheduled);
            }

            self.appendCell(row, Math.round(res.dist) + " meters");

            tblBody.appendChild(row);
        }

        tbl.appendChild(tblBody);
        results.appendChild(tbl);
    }
    xhr.send();
};
