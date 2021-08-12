load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "skycfg",
    srcs = [
        "module.go",
        "proto_api.go",
        "proto_enum.go",
        "proto_message.go",
        "proto_message_type.go",
        "proto_package.go",
        "proto_util.go",
    ],
    importpath = "github.com/stripe/skycfg/internal/go/skycfg",
    visibility = ["//:__subpackages__"],
    deps = [
        "@com_github_golang_protobuf//descriptor:go_default_library_gen",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@in_gopkg_yaml_v2//:yaml_v2",
        "@net_starlark_go//starlark",
        "@net_starlark_go//starlarkstruct",
        "@net_starlark_go//syntax",
        "@org_golang_google_protobuf//encoding/protojson",
        "@org_golang_google_protobuf//reflect/protoreflect",
        "@org_golang_google_protobuf//types/descriptorpb",
        "@org_golang_google_protobuf//types/known/durationpb",
        "@org_golang_google_protobuf//types/known/wrapperspb",
    ],
)

go_test(
    name = "skycfg_test",
    size = "small",
    srcs = [
        "proto_test.go",
        "skycfg_test.go",
    ],
    embed = [":skycfg"],
    deps = [
        "//:skycfg",
        "//internal/testdata/test_proto:test_proto_go_proto",
        "//internal/testdata/test_proto_gogo:test_proto_gogo_go_proto",
        "@com_github_gogo_protobuf//proto",
        "@com_github_gogo_protobuf//types",
        "@com_github_golang_protobuf//proto:go_default_library",
        "@com_github_kylelemons_godebug//pretty",
        "@net_starlark_go//resolve",
        "@net_starlark_go//starlark",
        "@net_starlark_go//syntax",
        "@org_golang_google_protobuf//types/descriptorpb",
        "@org_golang_google_protobuf//types/known/anypb",
        "@org_golang_google_protobuf//types/known/wrapperspb",
    ],
)
