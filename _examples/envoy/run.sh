#!/bin/bash
exec docker run -p9901:9901  -v`pwd`:/etc/envoy -it envoyproxy/envoy:v1.14-latest
