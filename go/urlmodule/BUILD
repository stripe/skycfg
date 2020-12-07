load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "urlmodule",
    srcs = ["urlmodule.go"],
    importpath = "github.com/stripe/skycfg/go/urlmodule",
    visibility = ["//visibility:public"],
    deps = [
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
    ],
)

go_test(
    name = "urlmodule_test",
    srcs = ["urlmodule_test.go"],
    embed = [":urlmodule"],
    deps = ["@net_starlark_go//starlark"],
)
