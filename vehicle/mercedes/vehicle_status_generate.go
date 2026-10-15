//go:build mercedes_proto

package mercedes

//go:generate protoc --go_out=. --go_opt=paths=source_relative vehicle_status.proto
