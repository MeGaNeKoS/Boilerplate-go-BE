package utils

import (
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

func TestProblemDetailSchemaRFC7807(t *testing.T) {
	s := registry.SchemaFromRef(problemDetailSchema.Ref)
	if v, ok := s.AdditionalProperties.(bool); !ok || !v {
		t.Fatalf("expected additionalProperties true, got %#v", s.AdditionalProperties)
	}

	p, ok := s.Properties["type"]
	if !ok || p == nil {
		t.Fatalf("type property missing")
	}
	if p.Type != huma.TypeString || p.Format != "uri-reference" {
		t.Fatalf("type property %#v", p)
	}
	if len(p.AnyOf) != 0 {
		t.Fatalf("unexpected AnyOf %#v", p.AnyOf)
	}
	if inst, ok := s.Properties["instance"]; !ok || inst == nil || inst.Format != "uri-reference" {
		t.Fatalf("instance format %#v", inst)
	}
}
