module github.com/openinfradev/tks-cluster-lcm

go 1.16

require (
	github.com/argoproj/argo-workflows/v3 v3.1.13
	github.com/google/uuid v1.3.0
	github.com/openinfradev/tks-contract v0.1.1-0.20210928021110-fe2b666327cc
	github.com/openinfradev/tks-info v0.0.0-20211015083247-48bc6cf48425
	github.com/openinfradev/tks-proto v0.0.6-0.20211015003551-ed8f9541f40d
	google.golang.org/grpc v1.41.0
	k8s.io/apimachinery v0.19.6
)

replace github.com/openinfradev/tks-cluster-lcm => ./

//replace github.com/openinfradev/tks-contract => ../tks-contract
//replace github.com/openinfradev/tks-proto => ../tks-proto
//replace github.com/openinfradev/tks-info => ../tks-info
