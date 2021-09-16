module github.com/openinfradev/tks-cluster-lcm

go 1.16

require (
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/jarcoal/httpmock v1.0.8
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/openinfradev/tks-contract v0.1.1-0.20210908032106-707946b426da
	github.com/openinfradev/tks-info v0.0.0-20210915080955-4650d3f62c02
	github.com/openinfradev/tks-proto v0.0.6-0.20210901093202-5e0db3fa3d4f
	github.com/sktelecom/tks-contract v0.1.0 // indirect
	golang.org/x/sys v0.0.0-20210910150752-751e447fb3d0 // indirect
	google.golang.org/grpc v1.40.0
)

replace github.com/openinfradev/tks-cluster-lcm => ./