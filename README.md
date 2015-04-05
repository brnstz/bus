# bus
MTA bus and train times

"Time To Go"?

# init

```
boot2docker up 
VBoxManage controlvm boot2docker-vm natpf1 "postgres-hello,tcp,127.0.0.1,5432,,5432"
VBoxManage controlvm boot2docker-vm natpf1 "redis-hello,tcp,127.0.0.1,6379,,6379"

docker run -d -p 5432:5432 postgres
docker run -d -p 6379:6379 redis
psql -U postgres -h $(boot2docker ip 2> /dev/null)

psql -U postgres -h 192.168.59.103 < models/schema.sql

go run cmds/stopload/main.go

# import / export
docker commit a22dee794ec8 bus-postgres:latest
docker save bus-postgres:latest > bus-postgres.tar

docker load < bus-redis.tar
docker run -d -p 5432:5432 bus-postgres
etc...
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

