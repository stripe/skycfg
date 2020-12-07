load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "hashmodule",
    srcs = ["hashmodule.go"],
    importpath = "github.com/stripe/skycfg/go/hashmodule",
    visibility = ["//visibility:public"],
    deps = [
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
    ],
)

go_test(
    name = "hashmodule_test",
    srcs = ["hashmodule_test.go"],
    embed = [":hashmodule"],
    deps = ["@net_starlark_go//starlark"],
)
