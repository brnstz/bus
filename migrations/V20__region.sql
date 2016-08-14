BEGIN;

    DROP SEQUENCE IF EXISTS region_seq;
    DROP MATERIALIZED VIEW IF EXISTS region;

    CREATE SEQUENCE region_seq;

    CREATE MATERIALIZED VIEW region AS
        SELECT 
            nextval('region_seq') AS id,
            'NYC'::text AS region_id,
            agency_id, 
            ST_MULTI(ST_UNION(stop.location)) AS locations
        FROM STOP
        GROUP BY agency_id;
    
    CREATE UNIQUE INDEX idx_unique_region ON region (id);
    CREATE INDEX idx_locations_region ON region USING gist(locations);

COMMIT;
