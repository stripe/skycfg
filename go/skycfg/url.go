package skycfg

import (
	"fmt"
	"net/url"

	"github.com/google/skylark"
)

// urlModule returns a Skylark module for URL helpers.
func urlModule() skylark.Value {
	return &skyModule{
		name: "url",
		attrs: skylark.StringDict{
			"encode_query": urlEncodeQuery(),
		},
	}
}

// urlEncodeQuery returns a Skylark function for encoding URL query strings.
//
//  def url.encode_query(query: dict[str, str]) -> str
//
// Query items will be encoded in Skylark iteration order.
func urlEncodeQuery() skylark.Callable {
	return skylark.NewBuiltin("url.encode_query", fnEncodeQuery)
}

func fnEncodeQuery(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var d *skylark.Dict
	if err := skylark.UnpackArgs(fn.Name(), args, kwargs, "query", &d); err != nil {
		return nil, err
	}

	urlVals := url.Values{}

	for _, itemPair := range d.Items() {
		key := itemPair[0]
		value := itemPair[1]

		keyStr, keyIsStr := key.(skylark.String)
		if !keyIsStr {
			return nil, fmt.Errorf("Key is not string: %+v", key)
		}

		valStr, valIsStr := value.(skylark.String)
		if !valIsStr {
			return nil, fmt.Errorf("Value is not string: %+v", value)
		}

		urlVals.Add(string(keyStr), string(valStr))
	}

	return skylark.String(urlVals.Encode()), nil
}
