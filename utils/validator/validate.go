package validator

import (
	"gopkg.in/go-playground/validator.v9"
	"regexp"
)

var validate = validator.New()

var (
	// NameRegex 名前正規表現
	NameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,32}$`)
	// ChannelRegex チャンネル名正規表現
	ChannelRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{1,20}$`)
	// PasswordRegex パスワード正規表現
	PasswordRegex = regexp.MustCompile(`^[\x20-\x7E]{10,32}$`)
	// TwitterIDRegex ツイッターIDの正規表現
	TwitterIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
	// PKCERegex PKCE文字列の正規表現
	PKCERegex = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")
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
	must(validate.RegisterValidation("name", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if len(s) > 0 {
			return NameRegex.MatchString(s)
		}
		return true
	}))
	must(validate.RegisterValidation("channel", func(fl validator.FieldLevel) bool {
		return ChannelRegex.MatchString(fl.Field().String())
	}))
	must(validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		return PasswordRegex.MatchString(fl.Field().String())
	}))
	must(validate.RegisterValidation("twitterid", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if len(s) > 0 {
			return TwitterIDRegex.MatchString(s)
		}
		return true
	}))
}

// ValidateStruct 構造体を検証します
func ValidateStruct(i interface{}) error {
	return validate.Struct(i)
}

// ValidateVar 値を検証します
func ValidateVar(i interface{}, tag string) error {
	return validate.Var(i, tag)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
