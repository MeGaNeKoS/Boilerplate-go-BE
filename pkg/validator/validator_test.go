package validator

import (
	"errors"
	"log"
	"reflect"
	"testing"
	"time"

	"github.com/bouk/monkey"
	ut "github.com/go-playground/universal-translator"
	v10 "github.com/go-playground/validator/v10"
)

type sample struct {
	Email string    `validate:"required,email"`
	Start time.Time `validate:"notBeforeNow"`
	End   time.Time `validate:"notAfterNow"`
}

func TestValidateStructSuccess(t *testing.T) {
	s := sample{Email: "a@b.com", Start: time.Now().Add(time.Second), End: time.Now()}
	if err := ValidateStruct(s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStructErrors(t *testing.T) {
	s := sample{Email: "bad", Start: time.Now().Add(-time.Hour), End: time.Now().Add(time.Hour)}
	err := ValidateStruct(s)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(ValidationErrors); !ok {
		t.Fatalf("wrong error type: %T", err)
	}
}
func TestValidationErrorsError(t *testing.T) {
	ve := NewValidationError("msg")
	if ve.Error() != "msg" {
		t.Fatalf("unexpected message %s", ve.Error())
	}
}

type stubFieldErr struct {
	tag    string
	field  string
	param  string
	actual string
}

func (s stubFieldErr) Tag() string { return s.tag }
func (s stubFieldErr) ActualTag() string {
	if s.actual == "" {
		return s.tag
	}
	return s.actual
}
func (s stubFieldErr) Namespace() string                 { return "" }
func (s stubFieldErr) StructNamespace() string           { return "" }
func (s stubFieldErr) Field() string                     { return s.field }
func (s stubFieldErr) StructField() string               { return "" }
func (s stubFieldErr) Value() interface{}                { return nil }
func (s stubFieldErr) Param() string                     { return s.param }
func (s stubFieldErr) Kind() reflect.Kind                { return reflect.String }
func (s stubFieldErr) Type() reflect.Type                { return reflect.TypeOf("") }
func (s stubFieldErr) Translate(ut ut.Translator) string { return "" }
func (s stubFieldErr) Error() string                     { return "" }

type stubFieldLevel struct{ val interface{} }

func (s stubFieldLevel) Top() reflect.Value      { return reflect.Value{} }
func (s stubFieldLevel) Parent() reflect.Value   { return reflect.Value{} }
func (s stubFieldLevel) Field() reflect.Value    { return reflect.ValueOf(s.val) }
func (s stubFieldLevel) FieldName() string       { return "" }
func (s stubFieldLevel) StructFieldName() string { return "" }
func (s stubFieldLevel) Param() string           { return "" }
func (s stubFieldLevel) GetTag() string          { return "" }
func (s stubFieldLevel) ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool) {
	return reflect.Value{}, 0, false
}
func (s stubFieldLevel) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) {
	return reflect.Value{}, 0, false
}
func (s stubFieldLevel) GetStructFieldOKAdvanced(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool) {
	return reflect.Value{}, 0, false
}
func (s stubFieldLevel) GetStructFieldOK2() (reflect.Value, reflect.Kind, bool, bool) {
	return reflect.Value{}, 0, false, false
}
func (s stubFieldLevel) GetStructFieldOKAdvanced2(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool, bool) {
	return reflect.Value{}, 0, false, false
}

func TestCustomErrorCases(t *testing.T) {
	cases := []struct{ tag, param string }{
		{"required", ""}, {"email", ""}, {"unique", ""}, {"notEmpty", ""},
		{"max", "5"}, {"min", "1"}, {"notBeforeNow", ""}, {"notAfterNow", ""},
		{"oneof", "a b"}, {"other", ""},
	}
	for _, c := range cases {
		err := customError(v10.ValidationErrors{stubFieldErr{tag: c.tag, field: "F", param: c.param}})
		if err == nil {
			t.Fatalf("expected error for %s", c.tag)
		}
	}
}

func TestCustomErrorInvalidValidation(t *testing.T) {
	e := &v10.InvalidValidationError{Type: reflect.TypeOf(0)}
	if !errors.Is(customError(e), e) {
		t.Fatalf("expected original error returned")
	}
}

func TestTimeValidatorsWrongType(t *testing.T) {
	if notAfterNow(stubFieldLevel{val: "no"}) {
		t.Fatal("expected false for invalid type")
	}
	if notBeforeNow(stubFieldLevel{val: "no"}) {
		t.Fatal("expected false for invalid type")
	}
}

func TestSetupRegisterError(t *testing.T) {
	prev := validate
	called := false
	patchFatal := monkey.Patch(log.Fatalf, func(string, ...interface{}) { called = true })
	monkey.PatchInstanceMethod(reflect.TypeOf(&v10.Validate{}), "RegisterValidation", func(*v10.Validate, string, v10.Func, ...bool) error {
		return errors.New("fail")
	})
	defer func() {
		monkey.UnpatchInstanceMethod(reflect.TypeOf(&v10.Validate{}), "RegisterValidation")
		patchFatal.Unpatch()
		validate = prev
	}()

	setup()
	if !called {
		t.Fatal("fatal not called")
	}
}
