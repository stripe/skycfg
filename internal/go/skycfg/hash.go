package skycfg

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"

	"go.starlark.net/starlark"
)

func HashModule() starlark.Value {
	return &Module{
		Name: "hash",
		Attrs: starlark.StringDict{
			"md5":    starlark.NewBuiltin("hash.md5", fnHashMd5),
			"sha1":   starlark.NewBuiltin("hash.sha1", fnHashSha1),
			"sha256": starlark.NewBuiltin("hash.sha256", fnHashSha256),
		},
	}
}

func fnHashMd5(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s starlark.String
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	h := md5.New()
	h.Write([]byte(string(s)))
	return starlark.String(fmt.Sprintf("%x", h.Sum(nil))), nil
}

func fnHashSha1(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s starlark.String
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	h := sha1.New()
	h.Write([]byte(string(s)))
	return starlark.String(fmt.Sprintf("%x", h.Sum(nil))), nil
}

func fnHashSha256(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var s starlark.String
	if err := starlark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(string(s)))
	return starlark.String(fmt.Sprintf("%x", h.Sum(nil))), nil
}
