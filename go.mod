module github.com/openinfradev/tks-cluster-lcm

go 1.16

require (
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/jarcoal/httpmock v1.0.8
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/openinfradev/tks-contract v0.1.1-0.20210915081037-2fef4d86b728
	github.com/openinfradev/tks-info v0.0.0-20210915080955-4650d3f62c02
	github.com/openinfradev/tks-proto v0.0.6-0.20210901093202-5e0db3fa3d4f
	golang.org/x/sys v0.0.0-20210910150752-751e447fb3d0 // indirect
	google.golang.org/grpc v1.40.0
)

replace github.com/openinfradev/tks-cluster-lcm => ./

replace github.com/openinfradev/tks-contract => ../tks-contract

replace github.com/openinfradev/tks-proto => ../tks-proto

replace github.com/openinfradev/tks-info => ../tks-info
