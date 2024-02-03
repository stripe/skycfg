workspace(name = "com_github_stripe_skycfg")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    integrity = "sha256-ZzSnGZk7G6Tr6YBuhThkOVqNOWitJ/nddZwZaz6zq+g=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.45.1/rules_go-v0.45.1.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.45.1/rules_go-v0.45.1.zip",
    ],
)


http_archive(
    name = "bazel_gazelle",
    integrity = "sha256-MpOL2hbmcABjA1R5Bj2dJMYO2o15/Uc5Vj9Q0zHLMgk=",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.35.0/bazel-gazelle-v0.35.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.35.0/bazel-gazelle-v0.35.0.tar.gz",
    ],
)

http_archive(
    name = "com_google_protobuf",
    sha256 = "6dd0f6b20094910fbb7f1f7908688df01af2d4f6c5c21331b9f636048674aebf",
    strip_prefix = "protobuf-3.14.0",
    urls = [
        "https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/protobuf-all-3.14.0.tar.gz",
    ],
    patches = [
        "//build:patches/protobuf_b8100.patch",
    ]
)

http_archive(
    name = "rules_proto",
    sha256 = "9850fcf6ad40fa348e6f13b2cfef4bb4639762f804794f2bf61d988f4ba0dae9",
    strip_prefix = "rules_proto-4.0.0-3.19.2-2",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/refs/tags/4.0.0-3.19.2-2.tar.gz",
    ],
)

load(
    "@rules_proto//proto:repositories.bzl",
    "rules_proto_dependencies",
    "rules_proto_toolchains",
)

load(
    "@io_bazel_rules_go//go:deps.bzl",
    "go_register_toolchains",
    "go_rules_dependencies",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

rules_proto_dependencies()

rules_proto_toolchains()

go_rules_dependencies()

load("//build:go_version.bzl", "GO_VERSION")
go_register_toolchains(go_version = GO_VERSION)

gazelle_dependencies()

# gazelle:repository_macro build/go_dependencies.bzl%go_dependencies
load("//build:go_dependencies.bzl", "go_dependencies")
go_dependencies()
