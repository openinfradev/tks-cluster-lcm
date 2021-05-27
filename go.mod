module github.com/sktelecom/tks-cluster-lcm

go 1.16

require (
	github.com/jarcoal/httpmock v1.0.8 // indirect
	github.com/sktelecom/tks-contract v0.1.1-0.20210421045537-2fe280d2c142
	github.com/sktelecom/tks-proto v0.0.4-0.20210517024623-b50937093731
	google.golang.org/grpc v1.38.0
)

replace github.com/sktelecom/tks-cluster-lcm v0.0.1 => ./
