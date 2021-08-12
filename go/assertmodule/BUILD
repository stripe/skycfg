load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "assertmodule",
    srcs = [
        "assert.go",
        "fail.go",
    ],
    importpath = "github.com/stripe/skycfg/go/assertmodule",
    visibility = ["//visibility:public"],
    deps = [
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
        "@net_starlark_go//syntax",
    ],
)

go_test(
    name = "assertmodule_test",
    size = "small",
    srcs = [
        "assert_test.go",
    ],
    embed = [":assertmodule"],
    deps = [
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
        "@net_starlark_go//syntax",
    ],
)
