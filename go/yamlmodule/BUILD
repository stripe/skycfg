load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "yamlmodule",
    srcs = [
        "json_write.go",
        "yamlmodule.go",
    ],
    importpath = "github.com/stripe/skycfg/go/yamlmodule",
    visibility = ["//visibility:public"],
    deps = [
        "@in_gopkg_yaml_v2//:yaml_v2",
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkjson",
        "@net_starlark_go//starlarkstruct",
    ],
)

go_test(
    name = "yamlmodule_test",
    srcs = ["yamlmodule_test.go"],
    embed = [":yamlmodule"],
    deps = ["@net_starlark_go//starlark"],
)
