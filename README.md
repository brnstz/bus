# bus
MTA bus and train times

"Time To Go"?

# init

```
# install postgres
# install postgres earthdistance and cube extensions
# install redis
# set local trust in /etc/postgresql/9.3/main/pg_hba.conf
# host    all             all             127.0.0.1/32            trust

# set env vars
export BUS_DB_ADDR=localhost:5432
export BUS_DB_USER=postgres
export BUS_REDIS_ADDR=localhost:6379
export BUS_TMP_DIR=/mnt/data/tmp
export BUS_API_ADDR=:8000
export MTA_BUS_TIME_API_KEY=<your bus time key>
export MTA_SUBWAY_TIME_API_KEY=<your subway time key>

# load initial schema
psql -U postgres -h $BUS_DB_HOST < schema/schema.sql

# build binaries
go install github.com/brnstz/bus/cmds/busapi

# Run it
$GOPATH/bin/busapi

# Hit it
# filter is optional, can be "subway" or "bus"
curl 'http://localhost:8000/api/v1/stops?lat=40.729183&lon=-73.95154&&miles=0.5&filter=subway' 
[
    {
        "direction_id": 0,
        "dist": 344.2649351427617,
        "headsign": "COURT SQ",
        "lat": 40.731352,
        "live": null,
        "lon": -73.954449,
        "route_id": "G",
        "scheduled": [
            {
                "desc": "",
                "time": "2015-07-10T17:43:30-04:00"
            },
            {
                "desc": "",
                "time": "2015-07-10T17:52:30-04:00"
            },
            {
                "desc": "",
                "time": "2015-07-10T17:59:30-04:00"
            }
        ],
        "station_type": "subway",
        "stop_id": "G26N",
        "stop_name": "Greenpoint Av"
    },
    {
        "direction_id": 1,
        "dist": 344.2649351427617,
        "headsign": "CHURCH AV",
        "lat": 40.731352,
        "live": null,
        "lon": -73.954449,
        "route_id": "G",
        "scheduled": [
            {
                "desc": "",
                "time": "2015-07-10T17:47:30-04:00"
            },
            {
                "desc": "",
                "time": "2015-07-10T17:55:30-04:00"
            },
            {
                "desc": "",
                "time": "2015-07-10T18:03:30-04:00"
            }
        ],
        "station_type": "subway",
        "stop_id": "G26S",
        "stop_name": "Greenpoint Av"
    }
]
```

# schema of transit files

```
==> agency.txt <==
agency_id,agency_name,agency_url,agency_timezone,agency_lang,agency_phone

==> calendar.txt <==
service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date

==> calendar_dates.txt <==
service_id,date,exception_type

==> routes.txt <==
route_id,agency_id,route_short_name,route_long_name,route_desc,route_type,route_url,route_color,route_text_color

==> shapes.txt <==
shape_id,shape_pt_lat,shape_pt_lon,shape_pt_sequence

==> stop_times.txt <==
trip_id,arrival_time,departure_time,stop_id,stop_sequence,pickup_type,drop_off_type

==> stops.txt <==
stop_id,stop_name,stop_desc,stop_lat,stop_lon,zone_id,stop_url,location_type,parent_station

==> trips.txt <==
route_id,service_id,trip_id,trip_headsign,direction_id,shape_id
```

# todo

  * ~~Load stops~~
  * ~~Load scheduled times / services~~
  * Load service exception days
  * ensure query for getServiceIdByDay is correct
  * Fix IP tables
  * ~~Document environment variables~~
  * ~~Automatically load new files from MTA~~
  * Build sample UI via web (get current location)
  * Build APIs:
    * ~~Given a lat/long, find a list of stops~~
    * Given a stop, find:
        * ~~A list of scheduled stop times (via database)~~
        * ~~A list of live stop times for bus~~
        * ~~A list of live stop times for subway~~
  * BUGS:
     * ~~API returns routes within the specified distance, but it chooses a
       random stop.~~
     * ~~Duplicate results from bus API (onward call vs. cur call? yes!)~~
     * ~~Needs caching~~

