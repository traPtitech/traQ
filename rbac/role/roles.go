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

	// 以下OAuth2のスコープと対応

	// ReadUser : 読み取り専用ユーザーロール
	ReadUser = gorbac.NewStdRole("read")
	// WriteUser : 書き込み専用ユーザーロール
	WriteUser = gorbac.NewStdRole("write")
	// PrivateReadUser : プライベートチャンネル読み取り専用ユーザーロール
	PrivateReadUser = gorbac.NewStdRole("private_read")
	// PrivateWriteUser : プライベートチャンネル書き込み専用ユーザーロール
	PrivateWriteUser = gorbac.NewStdRole("private_write")
	// ManageBot Botの管理権限
	ManageBot = gorbac.NewStdRole("manage_bot")
)

// SetRole : rbacに既定のロールをセットします
func SetRole(rbac *rbac.RBAC) {
	for r, ps := range map[*gorbac.StdRole][]gorbac.Permission{
		// 読み取り専用ユーザーのパーミッション
		ReadUser: {
			permission.GetChannel,

			permission.GetTopic,

			permission.GetMessage,

			permission.GetPin,

			permission.GetNotificationStatus,
			permission.ConnectNotificationStream,

			permission.GetUser,
			permission.GetMe,

			permission.GetClip,
			permission.GetClipFolder,

			permission.GetStar,

			permission.GetChannelVisibility,

			permission.GetUnread,

			permission.GetMutedChannels,

			permission.GetTag,

			permission.GetStamp,
			permission.GetMessageStamp,
			permission.GetMyStampHistory,

			permission.DownloadFile,

			permission.GetHeartbeat,

			permission.GetWebhook,

			permission.GetBot,
		},
		// 書き込み専用ユーザーのパーミッション
		WriteUser: {
			permission.CreateChannel,

			permission.EditTopic,

			permission.PostMessage,
			permission.EditMessage,
			permission.DeleteMessage,
			permission.ReportMessage,

			permission.CreatePin,
			permission.DeletePin,

			permission.ChangeNotificationStatus,
			permission.RegisterDevice,

			permission.EditMe,
			permission.ChangeMyIcon,

			permission.CreateClip,
			permission.DeleteClip,
			permission.CreateClipFolder,
			permission.PatchClipFolder,
			permission.DeleteClipFolder,

			permission.CreateStar,
			permission.DeleteStar,

			permission.DeleteUnread,

			permission.MuteChannel,
			permission.UnmuteChannel,

			permission.AddTag,
			permission.RemoveTag,
			permission.ChangeTagLockState,

			permission.CreateStamp,
			permission.AddMessageStamp,
			permission.RemoveMessageStamp,

			permission.EditStamp,

			permission.UploadFile,

			permission.PostHeartbeat,

			permission.InstallBot,
			permission.UninstallBot,
		},
		// プライベートチャンネル読み取り専用ユーザーのパーミッション
		PrivateReadUser: {}, // TODO
		// プライベートチャンネル書き込み専用ユーザーのパーミッション
		PrivateWriteUser: {}, // TODO
		// 一般ユーザーのパーミッション
		// ブラウザ(セッション)からの操作のみしか許可しない
		// ※ReadUser, WriteUser, PrivateReadUser, PrivateWriteUserのパーミッションを全て含む
		User: {
			permission.ChangeMyPassword,

			permission.GetMySessions,
			permission.DeleteMySessions,

			permission.GetMyTokens,
			permission.RevokeMyToken,
			permission.GetClients,
			permission.CreateClient,
			permission.EditMyClient,
			permission.DeleteMyClient,

			permission.CreateWebhook,
			permission.EditWebhook,
			permission.DeleteWebhook,

			permission.CreateBot,
			permission.EditBot,
			permission.DeleteBot,
			permission.ReissueBotToken,
		},
		// 管理者ユーザーのパーミッション
		// ※一般ユーザーのパーミッションを全て含む
		Admin: {
			permission.EditChannel,
			permission.DeleteChannel,
			permission.ChangeParentChannel,

			permission.GetMessageReports,

			permission.RegisterUser,
			permission.EditOtherUsers,

			permission.ChangeChannelVisibility,

			permission.EditTag,

			permission.AccessOthersWebhook,

			permission.EditStampName,
			permission.EditStampCreatedByOthers,
			permission.DeleteStamp,
			permission.DeleteFile,
		},
		// Botユーザーのパーミッション
		Bot: {
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

			permission.GetUser,
			permission.GetMe,

			permission.GetTag,
			permission.AddTag,
			permission.RemoveTag,
			permission.ChangeTagLockState,

			permission.GetStamp,
			permission.GetMessageStamp,
			permission.AddMessageStamp,
			permission.RemoveMessageStamp,

			permission.DownloadFile,
		},
		ManageBot: {
			permission.GetChannel,
			permission.GetUser,
			permission.GetMe,

			permission.GetWebhook,
			permission.CreateWebhook,
			permission.EditWebhook,
			permission.DeleteWebhook,

			permission.GetBot,
			permission.CreateBot,
			permission.EditBot,
			permission.DeleteBot,
			permission.ReissueBotToken,
			permission.InstallBot,
			permission.UninstallBot,
		},
	} {
		for _, p := range ps {
			_ = r.Assign(p)
		}
		_ = rbac.Add(r)
	}

	if err := rbac.SetParents(User.ID(), []string{ReadUser.ID(), WriteUser.ID(), PrivateReadUser.ID(), PrivateWriteUser.ID()}); err != nil {
		panic(err)
	}
	if err := rbac.SetParent(Admin.ID(), User.ID()); err != nil {
		panic(err)
	}
}
