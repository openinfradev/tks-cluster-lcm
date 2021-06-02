FROM golang:1.16.3

WORKDIR /go/src/tks-cluster-lcm
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...