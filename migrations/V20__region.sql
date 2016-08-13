BEGIN;

    CREATE SEQUENCE region_seq;

    CREATE MATERIALIZED VIEW region AS
        SELECT 
            nextval('region_seq') AS id,
            agency_id, 
            ST_MULTI(ST_UNION(stop.location)) AS locations
        FROM STOP
        GROUP BY agency_id;
    
    CREATE UNIQUE INDEX idx_unique_region ON region (id);

COMMIT;
