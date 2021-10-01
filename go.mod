module github.com/openinfradev/tks-cluster-lcm

go 1.16

require (
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0 // indirect
	github.com/jarcoal/httpmock v1.0.8
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/openinfradev/tks-contract v0.1.1-0.20210928021110-fe2b666327cc
	github.com/openinfradev/tks-info v0.0.0-20210928021400-117ed2408789
	github.com/openinfradev/tks-proto v0.0.6-0.20210924020717-178698d59e9d
	golang.org/x/net v0.0.0-20210928044308-7d9f5e0b762b // indirect
	google.golang.org/grpc v1.41.0
)

replace github.com/openinfradev/tks-cluster-lcm => ./

replace github.com/openinfradev/tks-contract => ../tks-contract

replace github.com/openinfradev/tks-proto => ../tks-proto

replace github.com/openinfradev/tks-info => ../tks-info
