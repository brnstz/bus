-- Create a new view using here_geometry
CREATE MATERIALIZED VIEW here_geometry

AS 

SELECT 
    sst.agency_id     AS agency_id,
    sst.route_id      AS route_id,
    sst.stop_id       AS stop_id,
    sst.service_id    AS service_id,
    sst.trip_id       AS trip_id,
    sst.arrival_sec   AS arrival_sec,
    sst.departure_sec AS departure_sec,
    sst.stop_sequence AS stop_sequence,

    stop.stop_name    AS stop_name,
    stop.direction_id AS direction_id,
    stop.headsign     AS stop_headsign,

    GEOMETRY(stop.location) AS location,

    route.route_type       AS route_type,
    route.route_color      AS route_color,
    route.route_text_color AS route_text_color,

    trip.headsign          AS trip_headsign

FROM scheduled_stop_time sst 

INNER JOIN stop ON 
    sst.agency_id = stop.agency_id AND
    sst.route_id  = stop.route_id  AND
    sst.stop_id   = stop.stop_id 

INNER JOIN route ON
    sst.agency_id = route.agency_id AND
    sst.route_id  = route.route_id

INNER JOIN trip ON
    sst.agency_id = sst.agency_id AND
    sst.trip_id   = trip.trip_id;

CREATE INDEX idx_location_here_geo ON here_geometry USING gist(location);
CREATE INDEX idx_service_id_here_geo ON here_geometry (service_id);

DROP MATERIALIZED VIEW here;
ALTER MATERIALIZED VIEW here_geometry RENAME TO here;
