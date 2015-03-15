CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

CREATE TYPE stop_type AS ENUM ('bus', 'subway');
CREATE TABLE stop (
    -- the unique id and name of the stop
    stop_id   TEXT NOT NULL,
    stop_name TEXT NOT NULL,

    -- direction of the stop, either 0 or 1
    direction_id INT NOT NULL,

    -- the direction this stop is going, e.g., "LI CITY QUEENS PLAZA" or "8 AV"
    headsign TEXT NOT NULL,

    -- the name of the route, e.g., "B62" or "G"
    route_id    TEXT NOT NULL,

    -- lat and lon converted into an earthdistance type
    location    EARTH NOT NULL,

    -- is this a bus or is this a subway?
    stype STOP_TYPE NOT NULL,

    UNIQUE(route_id, stop_id)
);

/*
VALUES('MTA_302255', 'MTA NYCT_B43', ll_to_earth(40.730251, -73.953064), 'bus');
          
select earth_distance(location, ll_to_earth(40.7, -73.95)) * 0.000621371192 from stop;

CREATE TYPE day_type AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');

CREATE TABLE service_days ( 
    service_id TEXT NOT NULL,
    route_id   TEXT NOT NULL,
    day        DAY_TYPE NOT NULL
);

CREATE TABLE scheduled_stop_times (
    service_id    TEXT NOT NULL,
    route_id      TEXT NOT NULL,
    trip_headsign TEXT NOT NULL,
    direction_id  INT  NOT NULL
);
*/
