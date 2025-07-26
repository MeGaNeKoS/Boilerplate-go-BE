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

func customError(err error) error {
	if err == nil {
		return nil
	}
	var castedObject validator.ValidationErrors
	if errors.As(err, &castedObject) {
		for _, err := range castedObject {
			switch err.Tag() {
			case "required":
				return NewValidationError(fmt.Sprintf("%s is required",
					err.Field()))
			case "email":
				return NewValidationError(fmt.Sprintf("%s is not valid email",
					err.Field()))
			case "unique":
				return NewValidationError(fmt.Sprintf("%s must unique value",
					err.Field()))
			case "notEmpty":
				return NewValidationError(fmt.Sprintf("%s can not be empty",
					err.Field()))
			case "max":
				return NewValidationError(fmt.Sprintf("%s value must be lower than %s", err.Field(), err.Param()))
			case "min":
				return NewValidationError(fmt.Sprintf("%s value must be grather than %s", err.Field(), err.Param()))
			case "notBeforeNow":
				return NewValidationError(fmt.Sprintf("%s must be greater than %s", err.Field(), time.Now().Format("2006-01-02")))
			case "notAfterNow":
				return NewValidationError(fmt.Sprintf("%s must be earlier than %s", err.Field(), time.Now().Format("2006-01-02")))
			case "oneof":
				return NewValidationError(fmt.Sprintf("%s must be one of = %s", err.Field(), err.Param()))
			default:
				return NewValidationError(fmt.Sprintf("%s validation error on %s tag", err.Field(), err.ActualTag()))
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
