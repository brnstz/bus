ALTER TABLE stop DROP CONSTRAINT stop_agency_id_route_id_stop_id_key;
CREATE UNIQUE INDEX stop_agency_id_route_id_stop_id_dir_id_key ON stop (agency_id, route_id, stop_id, direction_id);
