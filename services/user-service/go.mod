module user-service

go 1.23.8

require (
	common v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.64.1
	proto v0.0.0-00010101000000-000000000000
)

replace (
	common => ../../common
	proto => ../../proto
)

require (
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
