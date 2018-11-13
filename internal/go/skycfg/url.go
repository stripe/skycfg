package skycfg

import (
	"fmt"
	"net/url"

	"go.starlark.net/starlark"
)

// UrlModule returns a Starlark module for URL helpers.
func UrlModule() starlark.Value {
	return &Module{
		Name: "url",
		Attrs: starlark.StringDict{
			"encode_query": urlEncodeQuery(),
		},
	}
}

// urlEncodeQuery returns a Starlark function for encoding URL query strings.
//
//  def url.encode_query(query: dict[str, str]) -> str
//
// Query items will be encoded in starlark iteration order.
func urlEncodeQuery() starlark.Callable {
	return starlark.NewBuiltin("url.encode_query", fnEncodeQuery)
}

func fnEncodeQuery(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var d *starlark.Dict
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "query", &d); err != nil {
		return nil, err
	}

	urlVals := url.Values{}

	for _, itemPair := range d.Items() {
		key := itemPair[0]
		value := itemPair[1]

		keyStr, keyIsStr := key.(starlark.String)
		if !keyIsStr {
			return nil, fmt.Errorf("Key is not string: %+v", key)
		}

		valStr, valIsStr := value.(starlark.String)
		if !valIsStr {
			return nil, fmt.Errorf("Value is not string: %+v", value)
		}

		urlVals.Add(string(keyStr), string(valStr))
	}

	return starlark.String(urlVals.Encode()), nil
}
