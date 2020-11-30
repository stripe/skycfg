load("@rules_proto//proto:defs.bzl", "proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

# gazelle:exclude package.go
# gazelle:go_proto_compilers @io_bazel_rules_go//proto:gogofast_proto
# gazelle:resolve proto github.com/stripe/skycfg/internal/testdata/test_proto/test_proto_v2.proto //internal/testdata/test_proto
# gazelle:resolve proto github.com/gogo/protobuf/gogoproto/gogo.proto @gogo_special_proto//github.com/gogo/protobuf/gogoproto

proto_library(
    name = "test_proto_gogo",
    srcs = ["test_proto_gogo.proto"],
    visibility = ["//:__subpackages__"],
    deps = [
        "//internal/testdata/test_proto",
        "@com_google_protobuf//:duration_proto",
        "@gogo_special_proto//github.com/gogo/protobuf/gogoproto",
    ],
)

go_proto_library(
    name = "test_proto_gogo_go_proto",
    compilers = ["@io_bazel_rules_go//proto:gogofast_proto"],
    importpath = "github.com/stripe/skycfg/internal/testdata/test_proto_gogo",
    proto = ":test_proto_gogo",
    visibility = ["//:__subpackages__"],
    # keep
    deps = [
        "//internal/testdata/test_proto:test_proto_go_proto",
    ],
)
