package skycfg

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"

	"github.com/google/skylark"
)

func HashModule() skylark.Value {
	return &Module{
		Name: "hash",
		Attrs: skylark.StringDict{
			"md5":    skylark.NewBuiltin("hash.md5", fnHashMd5),
			"sha1":   skylark.NewBuiltin("hash.sha1", fnHashSha1),
			"sha256": skylark.NewBuiltin("hash.sha256", fnHashSha256),
		},
	}
}

func fnHashMd5(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var s skylark.String
	if err := skylark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	h := md5.New()
	h.Write([]byte(string(s)))
	return skylark.String(fmt.Sprintf("%x", h.Sum(nil))), nil
}

func fnHashSha1(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var s skylark.String
	if err := skylark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	h := sha1.New()
	h.Write([]byte(string(s)))
	return skylark.String(fmt.Sprintf("%x", h.Sum(nil))), nil
}

func fnHashSha256(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var s skylark.String
	if err := skylark.UnpackPositionalArgs(fn.Name(), args, kwargs, 1, &s); err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(string(s)))
	return skylark.String(fmt.Sprintf("%x", h.Sum(nil))), nil
}
