package validator

import (
	"gopkg.in/go-playground/validator.v9"
	"regexp"
)

var validate *validator.Validate

// Validator echo用バリデーター
type Validator struct {
	validator *validator.Validate
}

// Validate 構造体を検証します
func (v *Validator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}

// New echo用バリデーター
func New() *Validator {
	return &Validator{
		validator: validate,
	}
}

func init() {
	validate = validator.New()

	name := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
	validate.RegisterValidation("name", func(fl validator.FieldLevel) bool {
		return name.MatchString(fl.Field().String())
	})

	channel := regexp.MustCompile(`^[a-zA-Z0-9-_]{1,20}$`)
	validate.RegisterValidation("channel", func(fl validator.FieldLevel) bool {
		return channel.MatchString(fl.Field().String())
	})

	password := regexp.MustCompile(`^[\x20-\x7E]{10,32}$`)
	validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		return password.MatchString(fl.Field().String())
	})
}

// ValidateStruct 構造体を検証します
func ValidateStruct(i interface{}) error {
	return validate.Struct(i)
}
