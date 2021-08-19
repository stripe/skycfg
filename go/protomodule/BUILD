load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "protomodule",
    srcs = [
        "merge.go",
        "protomodule.go",
        "protomodule_enum.go",
        "protomodule_list.go",
        "protomodule_map.go",
        "protomodule_message.go",
        "protomodule_message_type.go",
        "protomodule_package.go",
        "type_conversions.go",
    ],
    importpath = "github.com/stripe/skycfg/go/protomodule",
    visibility = ["//visibility:public"],
    deps = [
        "//internal/go/skycfg",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
        "@net_starlark_go//syntax",
        "@org_golang_google_protobuf//encoding/protojson",
        "@org_golang_google_protobuf//encoding/prototext",
        "@org_golang_google_protobuf//proto",
        "@org_golang_google_protobuf//reflect/protoreflect",
        "@org_golang_google_protobuf//reflect/protoregistry",
        "@org_golang_google_protobuf//types/dynamicpb",
        "@org_golang_google_protobuf//types/known/anypb",
        "@org_golang_google_protobuf//types/known/wrapperspb",
    ],
)

go_test(
    name = "protomodule_test",
    srcs = [
        "protomodule_test.go",
    ],
    embed = [":protomodule"],
    deps = [
        "//:skycfg",
        "//internal/go/skycfg",
        "//internal/testdata/test_proto:test_proto_go_proto",
        "@net_starlark_go//resolve",
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
        "@net_starlark_go//syntax",
        "@org_golang_google_protobuf//reflect/protoregistry",
        "@org_golang_google_protobuf//types/descriptorpb",
        "@org_golang_google_protobuf//types/known/anypb",
    ],
)
