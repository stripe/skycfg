load("@rules_proto//proto:defs.bzl", "proto_library")
load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

# gazelle:exclude package.go

proto_library(
    name = "test_proto",
    srcs = [
        "test_proto_v2.proto",
        "test_proto_v3.proto",
    ],
    import_prefix = "github.com/stripe/skycfg",  # keep
    visibility = ["//:__subpackages__"],
    deps = [
        "@com_google_protobuf//:wrappers_proto",
        "@com_google_protobuf//:any_proto",
    ],
)

go_proto_library(
    name = "test_proto_go_proto",
    importpath = "github.com/stripe/skycfg/internal/testdata/test_proto",
    proto = ":test_proto",
    visibility = ["//:__subpackages__"],
    deps = [
        "@io_bazel_rules_go//proto/wkt:wrappers_go_proto",  # keep
        "@io_bazel_rules_go//proto/wkt:any_go_proto",  # keep
    ],
)
