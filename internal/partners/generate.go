package partners

// This will regenerate the *.pb.go files in transit_realtime/ and nyct_subway/
// (But those files are also stored in git)

//go:generate protoc --proto_path=./ --proto_path=../../../../../ --go_out=./ nyct_subway/nyct-subway.proto
//go:generate protoc --go_out=./ transit_realtime/gtfs-realtime.proto
