FROM golang:1.16.3-stretch AS builder
LABEL AUTHOR Seungkyu Ahn (seungkyua@gmail.com)

ARG PRV_GITHUB_TOKEN
RUN go env -w GOPRIVATE=github.com/openinfradev/tks-cluster-lcm
RUN git config --global url."https://x-access-token:${PRV_GITHUB_TOKEN}@github.com".insteadOf "https://github.com"

RUN mkdir -p /build
WORKDIR /build

COPY . .
RUN go mod tidy && go mod vendor
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server

RUN mkdir -p /dist
WORKDIR /dist
RUN cp /build/bin/server ./server



FROM golang:alpine3.13

RUN mkdir -p /app
WORKDIR /app

COPY --chown=0:0 --from=builder /dist /app/
EXPOSE 9111

ENTRYPOINT ["/app/server"]
CMD ["-port", "9110"]
