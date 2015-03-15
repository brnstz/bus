DROP schema PUBLIC cascade;
CREATE schema PUBLIC;

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
CREATE INDEX idx_location_stop ON stop USING gist(location);

CREATE TYPE day_type AS ENUM (
    'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 
    'saturday', 'sunday'
);

CREATE TABLE service_route_day (
    route_id    TEXT NOT NULL,
    service_id  TEXT NOT NULL,
    day         DAY_TYPE NOT NULL,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,

    UNIQUE(route_id, service_id, day, start_date, end_date)
);

CREATE TABLE service_route_exception (
    route_id        TEXT NOT NULL,
    service_id      TEXT NOT NULL,
    exception_date  DATE NOT NULL,

    UNIQUE(route_id, service_id, exception_date)
);

CREATE TABLE scheduled_stop_time (
    route_id       TEXT NOT NULL,
    stop_id        TEXT NOT NULL,
    service_id     TEXT NOT NULL,
    departure_sec  INT  NOT NULL,
    
    UNIQUE(route_id, stop_id, service_id, departure_sec)
);
-- CREATE INDEX idx_sst ON scheduled_stop_time (route_id, stop_id, service_id, departure_sec);
-- select route_id, service_id, max(start_day) from service_route_day where route_id = 'G' and day = 'monday' group by route_id, service_id;
