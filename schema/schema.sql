CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;

CREATE TYPE station_type AS ENUM ('bus', 'subway');
CREATE TABLE stop (
    stop_id     TEXT           NOT NULL,
    line        TEXT           NOT NULL,
    location    EARTH          NOT NULL,
    stype       STATION_TYPE   NOT NULL,

    UNIQUE(line, stop_id)
);

INSERT INTO stop (stop_id, line, location, stype) 
    VALUES('MTA_302255', 'MTA NYCT_B43', 
        ll_to_earth(40.730251, -73.953064), 'bus');
           
-- select earth_distance(location, ll_to_earth(40.7, -73.95)) * 0.000621371192 from stop;
