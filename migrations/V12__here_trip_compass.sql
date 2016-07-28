BEGIN;

    CREATE MATERIALIZED VIEW here_trip_v4

    AS 

    SELECT
        nextval('here_trip_seq') AS id,

        sst.agency_id     AS agency_id,
        sst.route_id      AS route_id,
        sst.stop_id       AS stop_id,
        sst.service_id    AS service_id,

        string_agg(sst.trip_id::text,       ',') AS trip_ids,
        string_agg(sst.arrival_sec::text,   ',') AS arrival_secs,
        string_agg(sst.departure_sec::text, ',') AS departure_secs,
        string_agg(sst.stop_sequence::text, ',') AS stop_sequences,
        string_agg(sst.next_stop_id::text, ',')  AS next_stop_ids,
        string_agg(
            degrees(ST_Azimuth(stop.location, next_stop.location))::text,
            ','
        ) AS compass_dirs,

        stop.stop_name    AS stop_name,
        stop.direction_id AS direction_id,
        stop.headsign     AS stop_headsign,
        stop.location     AS location,

        route.route_type       AS route_type,
        route.route_color      AS route_color,
        route.route_text_color AS route_text_color,

        trip.headsign          AS trip_headsign

    FROM scheduled_stop_time sst

    -- the actual stop
    INNER JOIN stop ON
        sst.agency_id = stop.agency_id AND
        sst.route_id  = stop.route_id  AND
        sst.stop_id   = stop.stop_id 

    -- the next stop
    INNER JOIN stop next_stop ON
        -- these will be the same
        sst.agency_id      = next_stop.agency_id AND
        sst.route_id       = next_stop.route_id  AND
        -- this will be different
        sst.next_stop_id   = next_stop.stop_id

    INNER JOIN route ON
        sst.agency_id = route.agency_id AND
        sst.route_id  = route.route_id

    INNER JOIN trip ON
        sst.agency_id = sst.agency_id AND
        sst.trip_id   = trip.trip_id

    WHERE
        -- FIXME: interestingly this check (and the entire last_stop field)
        -- should be unnecessary now that we are joining on the next stop

        -- backup condition while data in table is populating, this
        -- will include last stops
        sst.last_stop IS NULL OR

        -- typical condition once we have data
        sst.last_stop IS FALSE

    GROUP BY
    sst.agency_id, sst.route_id, sst.stop_id, sst.service_id,
    stop.stop_name, stop.direction_id, stop.headsign, stop.location,
    route.route_type, route.route_color, route.route_text_color, trip.headsign;

    CREATE INDEX idx_location_here_trip_v4 ON here_trip_v4 USING gist(location);
    CREATE INDEX idx_service_id_here_trip_v4 ON here_trip_v4 (service_id);
    CREATE UNIQUE INDEX idx_unique_here_trip_v4 ON here_trip_v4 (id);

    DROP  MATERIALIZED VIEW here_trip;
    ALTER MATERIALIZED VIEW here_trip_v4 RENAME TO here_trip;

COMMIT;
