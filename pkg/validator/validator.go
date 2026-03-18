package validator

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() { setup() }

func setup() {
	validate = validator.New()
	if err := validate.RegisterValidation("notBeforeNow", notBeforeNow); err != nil {
		log.Fatalf("register notBeforeNow: %v", err)
	}
	if err := validate.RegisterValidation("notAfterNow", notAfterNow); err != nil {
		log.Fatalf("register notAfterNow: %v", err)
	}
}

// ValidateStruct validates the struct using the registered validators and
// returns a formatted error if validation fails.
func ValidateStruct(obj interface{}) error {
	err := validate.Struct(obj)
	return customError(err)
}

// tagMessages maps validation tags to human-readable format strings.
// Use %s for the field name and %s for the parameter where applicable.
var tagMessages = map[string]string{
	"required": "%s is required",
	"email":    "%s is not a valid email",
	"unique":   "%s must have a unique value",
	"notEmpty": "%s cannot be empty",
	"max":      "%s value must be lower than %s",
	"min":      "%s value must be greater than %s",
	"oneof":    "%s must be one of: %s",
}

func customError(err error) error {
	if err == nil {
		return nil
	}
	var castedObject validator.ValidationErrors
	if errors.As(err, &castedObject) {
		for _, fe := range castedObject {
			switch fe.Tag() {
			case "notBeforeNow":
				return NewValidationError(fmt.Sprintf("%s must be greater than %s", fe.Field(), time.Now().Format("2006-01-02")))
			case "notAfterNow":
				return NewValidationError(fmt.Sprintf("%s must be earlier than %s", fe.Field(), time.Now().Format("2006-01-02")))
			default:
				if tmpl, ok := tagMessages[fe.Tag()]; ok {
					if fe.Param() != "" {
						return NewValidationError(fmt.Sprintf(tmpl, fe.Field(), fe.Param()))
					}
					return NewValidationError(fmt.Sprintf(tmpl, fe.Field()))
				}
				return NewValidationError(fmt.Sprintf("%s validation error on %s tag", fe.Field(), fe.ActualTag()))
			}
		}
	}
	return err
}

// NewValidationError wraps validation messages into a custom error type.
func NewValidationError(msg string) ValidationErrors {
	return ValidationErrors{errors.New(msg)}
}

// ValidationErrors represents an error encountered during validation.
type ValidationErrors struct {
	err error
}

// Error implements the error interface for ValidationErrors.
func (v ValidationErrors) Error() string {
	return v.err.Error()
}

func notAfterNow(fl validator.FieldLevel) bool {
	t, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	return !t.After(time.Now())
}

func notBeforeNow(fl validator.FieldLevel) bool {
	t, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	return !t.Before(time.Now())
}
