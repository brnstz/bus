BEGIN;

    CREATE SEQUENCE here_seq;
    CREATE SEQUENCE service_seq;
    CREATE SEQUENCE service_exception_seq;

    CREATE MATERIALIZED VIEW here_new

    AS 

    SELECT 
        nextval('here_seq') AS id,

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
        stop.location     AS location,

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

    CREATE INDEX idx_location_here_concur ON here_new USING gist(location);
    CREATE INDEX idx_service_id_here_concur ON here_new (service_id);
    CREATE UNIQUE INDEX idx_unique_here ON here_new (id);

    DROP MATERIALIZED VIEW here;
    ALTER MATERIALIZED VIEW here_new RENAME TO here;

    ---

    CREATE MATERIALIZED VIEW service_new AS
        SELECT DISTINCT ON (agency_id, service_id, day)
            nextval('service_seq') AS id,
            agency_id,
            service_id,
            day,
            start_date,
            end_date
        FROM service_route_day;
    CREATE INDEX idx_service_concur ON service_new (agency_id, service_id, day);
    CREATE UNIQUE INDEX idx_unique_service ON service_new (id);

    CREATE MATERIALIZED VIEW service_exception_new AS
        SELECT DISTINCT ON (agency_id, service_id, exception_date)
            nextval('service_exception_seq') AS id,
            agency_id,
            service_id,
            exception_date, 
            exception_type

        FROM service_route_exception;
    CREATE INDEX idx_service_exception_concur ON service_exception_new (agency_id, service_id, exception_date);
    CREATE UNIQUE INDEX idx_unique_service_exception ON service_exception_new (id);

COMMIT;
