load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "gogocompat",
    srcs = ["gogocompat.go"],
    importpath = "github.com/stripe/skycfg/gogocompat",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/go/skycfg",
        "@com_github_gogo_protobuf//proto",
        "@com_github_golang_protobuf//proto:go_default_library",
    ],
)

go_test(
    name = "gogocompat_test",
    size = "small",
    srcs = ["gogocompat_test.go"],
    embed = [":gogocompat"],
    deps = [
        "//internal/go/skycfg",
        "@com_github_gogo_protobuf//types",
        "@io_bazel_rules_go//proto/wkt:wrappers_go_proto",
        "@net_starlark_go//starlark",
    ],
)
