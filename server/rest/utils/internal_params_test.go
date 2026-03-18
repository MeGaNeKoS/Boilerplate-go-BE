package utils

import "testing"

type testInput struct {
	ID     int    `path:"id"`
	Secret string `header:"X-Secret" internal:"true"`
	Q      string `query:"q" internal:"true"`
}

func TestTrackInternalParams(t *testing.T) {
	ClearInternalParams()
	TrackInternalParams("op", testInput{})
	params := InternalParamsFor("op")
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d", len(params))
	}
	want := map[string]string{"X-Secret": "header", "q": "query"}
	for _, p := range params {
		if want[p.Name] != p.In {
			t.Fatalf("unexpected param %#v", p)
		}
		delete(want, p.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing params: %v", want)
	}
}

type pathFormCookieInput struct {
	ID string `path:"id" internal:"true"`
	F  string `form:"f" internal:"true"`
	C  string `cookie:"c" internal:"true"`
}

func TestTrackInternalParamsPointerAndLocations(t *testing.T) {
	ClearInternalParams()
	TrackInternalParams("op", (*pathFormCookieInput)(nil))
	params := InternalParamsFor("op")
	if len(params) != 3 {
		t.Fatalf("expected 3 params, got %d", len(params))
	}
	want := map[string]string{"id": "path", "f": "form", "c": "cookie"}
	for _, p := range params {
		if want[p.Name] != p.In {
			t.Fatalf("unexpected param %#v", p)
		}
		delete(want, p.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing params: %v", want)
	}
}

func TestTrackInternalParamsNil(t *testing.T) {
	ClearInternalParams()
	TrackInternalParams("op", nil)
	if params := InternalParamsFor("op"); len(params) != 0 {
		t.Fatalf("expected no params, got %v", params)
	}
}

func TestTrackInternalParamsNonStruct(t *testing.T) {
	ClearInternalParams()
	TrackInternalParams("op", 5)
	if params := InternalParamsFor("op"); len(params) != 0 {
		t.Fatalf("expected no params, got %v", params)
	}
}
