package validator

import (
	"gopkg.in/go-playground/validator.v9"
	"regexp"
)

var validate *validator.Validate

var (
	// NameRegex 名前正規表現
	NameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
	// ChannelRegex チャンネル名正規表現
	ChannelRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{1,20}$`)
	// PasswordRegex パスワード正規表現
	PasswordRegex = regexp.MustCompile(`^[\x20-\x7E]{10,32}$`)
)

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

	validate.RegisterValidation("name", func(fl validator.FieldLevel) bool {
		return NameRegex.MatchString(fl.Field().String())
	})

	validate.RegisterValidation("channel", func(fl validator.FieldLevel) bool {
		return ChannelRegex.MatchString(fl.Field().String())
	})

	validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		return PasswordRegex.MatchString(fl.Field().String())
	})
}

// ValidateStruct 構造体を検証します
func ValidateStruct(i interface{}) error {
	return validate.Struct(i)
}
