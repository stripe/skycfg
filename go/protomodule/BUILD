load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "protomodule",
    srcs = ["protomodule.go"],
    importpath = "github.com/stripe/skycfg/go/protomodule",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/go/skycfg",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
        "@org_golang_google_protobuf//encoding/protojson",
        "@org_golang_google_protobuf//encoding/prototext",
        "@org_golang_google_protobuf//proto",
        "@org_golang_google_protobuf//reflect/protoregistry",
        "@org_golang_google_protobuf//types/known/anypb",
    ],
)
