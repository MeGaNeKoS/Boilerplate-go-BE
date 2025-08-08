package helpers

import (
	"reflect"
	"strings"
)

// Param identifies a request parameter by name and location.
type Param struct {
	Name string
	In   string
}

var internalParams = map[string][]Param{}

// TrackInternalParams scans the given input struct type for fields tagged with
// `internal:"true"` and records any parameters so they can be stripped from
// public documentation. Supported locations are path, query, header, form, and
// cookie.
func TrackInternalParams(opID string, input any) {
	t := reflect.TypeOf(input)
	if t == nil {
		return
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Tag.Get("internal") != "true" {
			continue
		}
		var in, name string
		if v := f.Tag.Get("path"); v != "" {
			in, name = "path", v
		} else if v := f.Tag.Get("query"); v != "" {
			in, name = "query", strings.Split(v, ",")[0]
		} else if v := f.Tag.Get("header"); v != "" {
			in, name = "header", v
		} else if v := f.Tag.Get("form"); v != "" {
			in, name = "form", v
		} else if v := f.Tag.Get("cookie"); v != "" {
			in, name = "cookie", v
		}
		if name != "" {
			internalParams[opID] = append(internalParams[opID], Param{Name: name, In: in})
		}
	}
}

// InternalParamsFor returns any internal parameters registered for the
// provided operation ID.
func InternalParamsFor(opID string) []Param {
	return internalParams[opID]
}

// ClearInternalParams removes all recorded internal parameters. Intended for
// tests to reset the global registry.
func ClearInternalParams() {
	internalParams = map[string][]Param{}
}
