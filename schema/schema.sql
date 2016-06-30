DROP schema PUBLIC cascade;
CREATE schema PUBLIC;

CREATE EXTENSION IF NOT EXISTS postgis;

-- route contains selected fields from 
-- https://developers.google.com/transit/gtfs/reference#routestxt
CREATE TABLE route (
    agency_id           TEXT NOT NULL,

    -- route_id is the unique id of this route
    route_id            TEXT NOT NULL,

    -- route_type is an id code for whether it's a subway, bus, etc. 
    -- (e.g., subway=1, bus=3, see link above for full doc)
    route_type          INT NOT NULL,

    -- route_color and route_text_color are the hex values for the background
    -- and foreground color of the route, respectively (e.g., 000000)
    route_color         TEXT NOT NULL,
    route_text_color    TEXT NOT NULL,

    UNIQUE(agency_id, route_id)
);

CREATE TABLE stop (
    -- the unique id and name of the stop
    agency_id TEXT NOT NULL,
    stop_id   TEXT NOT NULL,
    stop_name TEXT NOT NULL,

    -- direction of the stop, either 0 or 1
    direction_id INT NOT NULL,

    -- the direction this stop is going, e.g., "LI CITY QUEENS PLAZA" or "8 AV"
    headsign TEXT NOT NULL,

    -- the name of the route, e.g., "B62" or "G"
    route_id    TEXT NOT NULL,

    -- lat and lon converted into an earthdistance type
    location    GEOGRAPHY(POINT, 4326) NOT NULL,

    UNIQUE(agency_id, route_id, stop_id)
);
CREATE INDEX idx_location_stop ON stop USING gist(location);

CREATE TYPE day_type AS ENUM ('monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday');

CREATE TABLE service_route_day (
    agency_id   TEXT NOT NULL,
    route_id    TEXT NOT NULL,
    service_id  TEXT NOT NULL,
    day         DAY_TYPE NOT NULL,
    start_date  DATE NOT NULL,
    end_date    DATE NOT NULL,

    UNIQUE(agency_id, route_id, service_id, day, start_date, end_date)
);

CREATE TABLE trip (
    agency_id    TEXT NOT NULL,
    route_id     TEXT NOT NULL,
    trip_id      TEXT NOT NULL,
    service_id   TEXT NOT NULL,
    shape_id     TEXT NOT NULL,
    headsign     TEXT NOT NULL,
    direction_id INT NOT NULL,

    UNIQUE(agency_id, route_id, trip_id)
);

CREATE TABLE shape (
    agency_id   TEXT NOT NULL,
    shape_id    TEXT NOT NULL,
    location    GEOGRAPHY(POINT, 4326) NOT NULL,
    seq         INT NOT NULL,

    UNIQUE(agency_id, shape_id, seq)
);

CREATE TABLE service_route_exception (
    agency_id       TEXT NOT NULL,
    route_id        TEXT NOT NULL,
    service_id      TEXT NOT NULL,
    exception_date  DATE NOT NULL,
    exception_type  INT NOT NULL,

    UNIQUE(agency_id, route_id, service_id, exception_date)
);

CREATE TABLE scheduled_stop_time (
    agency_id      TEXT NOT NULL,
    route_id       TEXT NOT NULL,
    stop_id        TEXT NOT NULL,
    service_id     TEXT NOT NULL,
    trip_id        TEXT NOT NULL,
    arrival_sec    INT  NOT NULL,
    departure_sec  INT  NOT NULL,
    stop_sequence  INT  NOT NULL,
    
    UNIQUE(agency_id, route_id, stop_id, service_id, trip_id)
);

-- route_shape contains "all" shape_ids need to draw a route. This should be
-- the "biggest" shape (most points) for each agency_id + route_id +
-- trip.headsign + direction_id combination. 
CREATE TABLE route_shape (
    agency_id       TEXT NOT NULL,
    route_id        TEXT NOT NULL,
    direction_id    INT NOT NULL,
    headsign        TEXT NOT NULL,
    shape_id        TEXT NOT NULL,

    UNIQUE(agency_id, route_id, direction_id, headsign)
);


-- route_trip contains the "biggest" (most stops) trip_id for reach
-- agency_id + route_id + direction_id + trip.headsign combination
CREATE TABLE route_trip (
    agency_id      TEXT NOT NULL,
    route_id       TEXT NOT NULL,
    direction_id   INT NOT NULL,
    headsign       TEXT NOT NULL,
    trip_id        TEXT NOT NULL,

    UNIQUE(agency_id, route_id, direction_id, headsign)
);
