CREATE MATERIALIZED VIEW region AS
    SELECT 
        agency_id, 
        ST_MULTI(ST_UNION(stop.location)) AS locations
    FROM STOP
    GROUP BY agency_id;
