-- Rethinking the service_route and service_route_exception tables. 
-- Probably don't need the route. Create materialized views to 
-- test out a possible final solution in the loader later.

CREATE MATERIALIZED VIEW service AS
    SELECT DISTINCT ON (agency_id, service_id, day) 
        agency_id,
        service_id, 
        day, 
        start_date,
        end_date

    FROM service_route_day;
CREATE INDEX idx_service ON service (agency_id, service_id, day);

CREATE MATERIALIZED VIEW service_exception AS
    SELECT DISTINCT ON (agency_id, service_id, exception_date)
        agency_id,
        service_id,
        exception_date, 
        exception_type

    FROM service_route_exception;

CREATE INDEX idx_service_exception ON service_exception (agency_id, service_id, exception_date);
