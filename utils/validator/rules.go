package validator

import (
	"regexp"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// RequiredIfValid utils/optional系の値がvalid時に空の値を弾く
//
// 仕組み:
// utils/optional系の値は sql.Valuer を実装している
// Valid: false の場合、nilがvalidationされるので通る
// Valid: true かつ空の値の場合、空の値がvalidationされるので通らない
//
// 分かりやすいように & このコメントを書くため名前を付けている
var RequiredIfValid = vd.NilOrNotEmpty

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

// UserGroupNameRule ユーザーグループ名バリデーションルール
var UserGroupNameRule = []vd.Rule{
	vd.Match(regexp.MustCompile(`^[^@＠#＃:： 　]*$`)).Error("must not contain [@＠#＃:：] and spaces"),
	vd.RuneLength(1, 30),
}

// UserGroupNameRuleRequired ユーザーグループ名バリデーションルール with Required
var UserGroupNameRuleRequired = append([]vd.Rule{
	vd.Required,
}, UserGroupNameRule...)

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

// StampPaletteNameRule スタンプパレット名バリデーションルール
var StampPaletteNameRule = []vd.Rule{
	vd.RuneLength(1, 30),
}

// StampPaletteNameRuleRequired スタンプパレット名バリデーションルール with Required
var StampPaletteNameRuleRequired = append([]vd.Rule{
	vd.Required,
}, StampPaletteNameRule...)

// StampPaletteDescriptionRule スタンプパレット説明バリデーションルール
var StampPaletteDescriptionRule = []vd.Rule{
	vd.RuneLength(0, 1000),
}

// StampPaletteStampsRule スタンプパレット内スタンプバリデーションルール
var StampPaletteStampsRule = []vd.Rule{
	vd.Length(0, 200),
}

// StampPaletteStampsRuleNotNil スタンプパレット内スタンプバリデーションルール with NotNil
var StampPaletteStampsRuleNotNil = append([]vd.Rule{
	vd.NotNil,
}, StampPaletteStampsRule...)

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
	vd.RuneLength(0, 1000),
}
