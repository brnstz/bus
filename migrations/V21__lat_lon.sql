ALTER TABLE shape ADD COLUMN lat double precision;
ALTER TABLE shape ADD COLUMN lon double precision;

ALTER TABLE fake_shape ADD COLUMN lat double precision;
ALTER TABLE fake_shape ADD COLUMN lon double precision;

ALTER TABLE scheduled_stop_time ADD COLUMN next_stop_lat double precision;
ALTER TABLE scheduled_stop_time ADD COLUMN next_stop_lon double precision;

ALTER TABLE stop ADD COLUMN lat double precision;
ALTER TABLE stop ADD COLUMN lon double precision;
