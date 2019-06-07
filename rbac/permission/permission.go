package permission

import "github.com/traPtitech/traQ/rbac"

var List = rbac.Permissions{}

func init() {
	l := []rbac.Permission{
		GetWebhook,
		CreateWebhook,
		EditWebhook,
		DeleteWebhook,
		AccessOthersWebhook,

		GetBot,
		CreateBot,
		EditBot,
		DeleteBot,
		ReissueBotToken,
		InstallBot,
		UninstallBot,

		CreateChannel,
		GetChannel,
		EditChannel,
		DeleteChannel,
		ChangeParentChannel,
		GetTopic,
		EditTopic,
		GetChannelVisibility,
		ChangeChannelVisibility,

		GetMyTokens,
		RevokeMyToken,
		GetClients,
		CreateClient,
		EditMyClient,
		DeleteMyClient,

		GetClip,
		CreateClip,
		DeleteClip,
		GetClipFolder,
		CreateClipFolder,
		PatchClipFolder,
		DeleteClipFolder,

		UploadFile,
		DownloadFile,
		DeleteFile,

		GetHeartbeat,
		PostHeartbeat,

		GetMessage,
		PostMessage,
		EditMessage,
		DeleteMessage,
		ReportMessage,
		GetMessageReports,

		GetMutedChannels,
		MuteChannel,
		UnmuteChannel,

		GetNotificationStatus,
		ChangeNotificationStatus,
		ConnectNotificationStream,
		RegisterDevice,

		GetPin,
		CreatePin,
		DeletePin,

		GetMySessions,
		DeleteMySessions,

		GetStamp,
		CreateStamp,
		EditStamp,
		EditStampName,
		EditStampCreatedByOthers,
		DeleteStamp,
		GetMessageStamp,
		AddMessageStamp,
		RemoveMessageStamp,
		GetMyStampHistory,

		GetStar,
		CreateStar,
		DeleteStar,

		GetUnread,
		DeleteUnread,

		GetUser,
		RegisterUser,
		GetMe,
		EditMe,
		ChangeMyIcon,
		ChangeMyPassword,
		EditOtherUsers,

		GetTag,
		AddTag,
		RemoveTag,
		ChangeTagLockState,
		EditTag,
	}
	for _, p := range l {
		List.Add(p)
	}
}
