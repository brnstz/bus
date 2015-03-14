CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

CREATE TYPE station_type AS ENUM ('bus', 'subway');
CREATE TABLE stop (
    stop_id     TEXT           NOT NULL,
    route_id    TEXT           NOT NULL,
    location    EARTH          NOT NULL,
    stype       STATION_TYPE   NOT NULL,

    UNIQUE(route_id, stop_id)
);

    VALUES('MTA_302255', 'MTA NYCT_B43', 
        ll_to_earth(40.730251, -73.953064), 'bus');
           
-- select earth_distance(location, ll_to_earth(40.7, -73.95)) * 0.000621371192 from stop;

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
