package partners

//go:generate protoc --proto_path=./ --proto_path=../../../../../ --go_out=./ nyct_subway/nyct-subway.proto
//go:generate protoc --go_out=./ transit_realtime/gtfs-realtime.proto
