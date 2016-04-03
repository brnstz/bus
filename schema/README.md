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
