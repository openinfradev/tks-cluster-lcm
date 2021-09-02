module github.com/openinfradev/tks-cluster-lcm

go 1.16

require (
	github.com/jarcoal/httpmock v1.0.8
	github.com/openinfradev/tks-contract v0.1.1-0.20210902134454-132819708ac3
	github.com/openinfradev/tks-proto v0.0.6-0.20210901093202-5e0db3fa3d4f
	google.golang.org/grpc v1.38.0
)

replace github.com/openinfradev/tks-cluster-lcm v0.0.1 => ./
