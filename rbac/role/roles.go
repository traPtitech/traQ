package role

import (
	"github.com/mikespook/gorbac"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
)

var (
	// Admin : 管理者ユーザーロール
	Admin = gorbac.NewStdRole("admin")
	// User : 一般ユーザーロール
	User = gorbac.NewStdRole("user")
	// Bot : Botユーザーロール
	Bot = gorbac.NewStdRole("bot")
)

// SetRole : rbacに既定のロールをセットします
func SetRole(rbac *rbac.RBAC) {
	// 一般ユーザーのパーミッション
	for _, p := range []gorbac.Permission{
		permission.CreateChannel,
		permission.GetChannel,

		permission.GetTopic,
		permission.EditTopic,

		permission.GetMessage,
		permission.PostMessage,
		permission.EditMessage,
		permission.DeleteMessage,

		permission.GetPin,
		permission.CreatePin,
		permission.DeletePin,

		permission.GetNotificationStatus,
		permission.ChangeNotificationStatus,
		permission.ConnectNotificationStream,
		permission.RegisterDevice,

		permission.GetUser,
		permission.GetMe,
		permission.EditMe,
		permission.ChangeMyIcon,

		permission.GetClip,
		permission.CreateClip,
		permission.DeleteClip,

		permission.GetStar,
		permission.CreateStar,
		permission.DeleteStar,

		permission.GetChannelVisibility,

		permission.GetUnread,
		permission.DeleteUnread,

		permission.GetTag,
		permission.AddTag,
		permission.RemoveTag,
		permission.ChangeTagLockState,

		permission.GetStamp,
		permission.CreateStamp,
		permission.GetMessageStamp,
		permission.AddMessageStamp,
		permission.RemoveMessageStamp,

		permission.UploadFile,
		permission.DownloadFile,

		permission.GetHeartbeat,
		permission.PostHeartbeat,

		permission.GetWebhook,
		permission.CreateWebhook,
		permission.EditWebhook,
		permission.DeleteWebhook,
	} {
		if err := User.Assign(p); err != nil {
			panic(err)
		}
	}

	// 管理者ユーザーのパーミッション
	// ※一般ユーザーのパーミッションを全て含む
	for _, p := range []gorbac.Permission{
		permission.EditChannel,
		permission.DeleteChannel,

		permission.RegisterUser,

		permission.ChangeChannelVisibility,

		permission.EditStamp,
		permission.DeleteStamp,
		permission.DeleteFile,
	} {
		if err := Admin.Assign(p); err != nil {
			panic(err)
		}
	}

	for _, r := range []gorbac.Role{
		Bot,
		User,
		Admin,
	} {
		if err := rbac.Add(r); err != nil {
			panic(err)
		}
	}
	if err := rbac.SetParent(Admin.ID(), User.ID()); err != nil {
		panic(err)
	}
}
