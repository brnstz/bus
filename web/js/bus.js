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

Bus.prototype.appendTime = function(row, res, type) {
    if (res[type] && res[type].length > 0) {
        var mytext = "Next " + type + ": " + res[type][0]["desc"];
        var mytime = new Date(res[type][0]["time"]);

        if (mytime.getFullYear() > 0) {
            var diff = Math.abs(new Date() - mytime);
            console.log(diff);
            mytext = mytext + " " + mytime.toTimeString();
        }

        this.appendCell(row, mytext);
    } else {
        this.appendCell(row, "")
    }
};

Bus.prototype.getTrips = function() {
    var self = this;
    var xhr = new XMLHttpRequest();
    var url = '/api/v2/stops?lat=' + this.lat +
              '&lon='			   + this.lon +
              '&filter='           + this.filter +
              '&miles='	           + this.miles;

    xhr.open('GET', url);
    xhr.onload = function(e) {
        var data = JSON.parse(this.response);
        var tbl     = document.createElement("table");
        tbl.setAttribute("class", "table");
        var tblBody = document.createElement("tbody");
        var results = document.getElementById("results");

        if (results.childNodes.length > 0) {
            results.removeChild(results.childNodes[0]);
        }

        for (var i = 0; i < data.length; i++) {
            var res = data[i];
            var row = document.createElement("tr");	

            self.appendCell(row, res["route_id"]);
            self.appendCell(row, res["stop_name"]);
            self.appendCell(row, res["headsign"]);
            self.appendTime(row, res, "live");
            self.appendTime(row, res, "scheduled");

            self.appendCell(row, Math.round(res["dist"]) + " meters");

            tblBody.appendChild(row);
        }		

        tbl.appendChild(tblBody);
        results.appendChild(tbl);
    }
    xhr.send();
};
