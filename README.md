# tks-cluster-lcm

[![Go Report Card](https://goreportcard.com/badge/github.com/openinfradev/tks-cluster-lcm?style=flat-square)](https://goreportcard.com/report/github.com/openinfradev/tks-cluster-lcm)
[![Go Reference](https://pkg.go.dev/badge/github.com/openinfradev/tks-cluster-lcm.svg)](https://pkg.go.dev/github.com/openinfradev/tks-cluster-lcm)
[![Release](https://img.shields.io/github/release/sktelecom/tks-cluster-lcm.svg?style=flat-square)](https://github.com/openinfradev/tks-cluster-lcm/releases/latest)

TKS는 Taco Kubernetes Service의 약자로, SK Telecom이 만든 GitOps기반의 서비스 시스템을 의미합니다. 그 중 Tks-cluster-lcm은 tks cluster 및 기반 서비스들의 생성, 조회 및 삭제 등 전반적인 lifecycle을 관리하는 서비스이며, 다른 tks service들과 gRPC 기반으로 통신합니다. gRPC 호출을 위한 proto 파일은 [tks-proto](https://github.com/openinfradev/tks-proto)에서 확인할 수 있습니다.

## Quick Start

### Prerequisite
* docker 20.x 설치 (docker로 구동할 경우)
* tks-info 설치
  * tks-info: https://github.com/openinfradev/tks-info
* decapod component 설치
  * decapod-bootstrap: https://github.com/openinfradev/decapod-bootstrap
  * decapod document: https://openinfradev.github.io/decapod-docs

### 서비스 구동 (For go developers)

```
$ CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/tks-cluster-lcm ./cmd/server/
$ bin/tks-cluster-lcm -port 9110
```

### 서비스 구동 (For docker users)
```
$ docker pull sktcloud/tks-cluster-lcm
$ docker run --name tks-cluster-lcm -p 9110:9110 -d \
   sktcloud/tks-cluster-lcm server -port 9110
```

### gRPC API 호출 예제 (golang)

```
import (
  "google.golang.org/grpc"
  pb "github.com/openinfradev/tks-proto/tks_pb"
  "google.golang.org/protobuf/encoding/protojson"
  "google.golang.org/protobuf/types/known/timestamppb"
  ...
)

  func YOUR_FUNCTION(YOUR_PARAMS...) {
    var conn *grpc.ClientConn
    tksInfoUrl = viper.GetString("tksInfoUrl")
    if tksInfoUrl == "" {
      fmt.Println("You must specify tksInfoUrl at config file")
      os.Exit(1)
    }
    conn, err := grpc.Dial(tksInfoUrl, grpc.WithInsecure())
    if err != nil {
      log.Fatalf("Couldn't connect to tks-info: %s", err)
    }
    defer conn.Close()

    client := pb.NewClusterInfoServiceClient(conn)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()
    data := pb.GetClustersRequest{}
    data.ContractId = viper.GetString("contractId")

    m := protojson.MarshalOptions{
      Indent:        "  ",
      UseProtoNames: true,
    }
    jsonBytes, _ := m.Marshal(&data)

    r, err := client.GetClusters(ctx, &data)
    if err != nil {
      fmt.Println(err)
    } else {
      if len(r.Clusters) == 0 {
        fmt.Println("No cluster exists for specified contract!")
      } else {
        /* print cluster info from 'r' */


 
      }
    }
  }
```
