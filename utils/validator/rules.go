package validator

import (
	"regexp"

	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

// PasswordRule パスワードバリデーションルール
var PasswordRule = []vd.Rule{
	is.PrintableASCII,
	vd.RuneLength(10, 32),
}

// PasswordRuleRequired パスワードバリデーションルール with Required
var PasswordRuleRequired = append([]vd.Rule{
	vd.Required,
}, PasswordRule...)

// UserNameRule ユーザー名バリデーションルール
var UserNameRule = []vd.Rule{
	vd.Match(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)).Error("must contain [a-zA-Z0-9_-] only"),
	vd.RuneLength(1, 32),
}

// UserNameRuleRequired ユーザー名バリデーションルール with Required
var UserNameRuleRequired = append([]vd.Rule{
	vd.Required,
}, UserNameRule...)

// BotUserNameRule BOTユーザー名バリデーションルール
var BotUserNameRule = []vd.Rule{
	vd.Match(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)).Error("must contain [a-zA-Z0-9_-] only"),
	vd.RuneLength(1, 20),
}

// BotUserNameRuleRequired BOTユーザー名バリデーションルール with Required
var BotUserNameRuleRequired = append([]vd.Rule{
	vd.Required,
}, BotUserNameRule...)

// ChannelNameRule チャンネル名バリデーションルール
var ChannelNameRule = []vd.Rule{
	vd.Match(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)).Error("must contain [a-zA-Z0-9_-] only"),
	vd.RuneLength(1, 20),
}

// ChannelNameRuleRequired チャンネル名バリデーションルール with Required
var ChannelNameRuleRequired = append([]vd.Rule{
	vd.Required,
}, ChannelNameRule...)

// StampNameRule スタンプ名バリデーションルール
var StampNameRule = []vd.Rule{
	vd.Match(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)).Error("must contain [a-zA-Z0-9_-] only"),
	vd.RuneLength(1, 32),
}

// StampNameRuleRequired スタンプ名バリデーションルール with Required
var StampNameRuleRequired = append([]vd.Rule{
	vd.Required,
}, StampNameRule...)

// TwitterIDRule TwitterIDバリデーションルール
var TwitterIDRule = []vd.Rule{
	vd.Match(regexp.MustCompile(`^[a-zA-Z0-9_]+$`)).Error("must contain [a-zA-Z0-9_] only"),
	vd.RuneLength(1, 15),
}

// ClipFolderNameRule クリップフォルダー名バリデーションルール
var ClipFolderNameRule = []vd.Rule{
	vd.RuneLength(1, 30),
}

// ClipFolderNameRuleRequired クリップフォルダー名バリデーションルール with Required
var ClipFolderNameRuleRequired = append([]vd.Rule{
	vd.Required,
}, ClipFolderNameRule...)

// ClipFolderDescriptionRule クリップフォルダーの説明バリデーションルール
var ClipFolderDescriptionRule = []vd.Rule{
	vd.Length(0, 1000),
}
