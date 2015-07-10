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
export BUS_DB_HOST=127.0.0.1
export BUS_REDIS_HOST=127.0.0.1
export MTA_BUS_TIME_API_KEY=<your bus time key>
export MTA_SUBWAY_TIME_API_KEY=<your subway time key>

# load initial schema
psql -U postgres -h $BUS_DB_HOST < schema/schema.sql

# build binaries
go install github.com/brnstz/bus/cmds/stopload
go install github.com/brnstz/bus/cmds/busapi

go run cmds/stopload/main.go
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
  * Document environment variables
  * Automatically load new files from MTA
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

