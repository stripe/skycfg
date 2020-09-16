FROM golang:1.15
COPY . /src
WORKDIR /src
#RUN go run main.go envoy.cfg > envoy.json

FROM envoyproxy/envoy-dev:8fc484325df6b0c95d9c0e2cb6161d25851574f9
COPY --from=0 /src/envoy.json /etc/envoy/envoy.json
