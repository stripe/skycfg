#!/bin/bash
exec docker run -i -p10001:10001 -p10000:10000 -p9901:9901  -v`pwd`:/etc/envoy -it envoyproxy/envoy:v1.14-latest
