CREATE TABLE fake_shape (
    agency_id       TEXT NOT NULL,
    route_id        TEXT NOT NULL,
    direction_id    INT NOT NULL,
    headsign        TEXT NOT NULL,
    seq             INT NOT NULL,
    location        GEOMETRY NOT NULL,

    UNIQUE(agency_id, route_id, direction_id, headsign, seq)
);
