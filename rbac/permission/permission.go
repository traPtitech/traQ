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
		AccessOthersBot,

		BotActionJoinChannel,
		BotActionLeaveChannel,

		CreateChannel,
		GetChannel,
		EditChannel,
		DeleteChannel,
		ChangeParentChannel,
		EditChannelTopic,

		GetMyTokens,
		RevokeMyToken,
		GetClients,
		CreateClient,
		EditMyClient,
		DeleteMyClient,
		ManageOthersClient,

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

		GetChannelSubscription,
		EditChannelSubscription,
		ConnectNotificationStream,
		RegisterFCMDevice,

		CreateMessagePin,
		DeleteMessagePin,

		GetMySessions,
		DeleteMySessions,

		GetMyExternalAccount,
		EditMyExternalAccount,

		GetStamp,
		CreateStamp,
		EditStamp,
		EditStampCreatedByOthers,
		DeleteStamp,
		AddMessageStamp,
		RemoveMessageStamp,
		GetMyStampHistory,

		GetChannelStar,
		EditChannelStar,

		GetUnread,
		DeleteUnread,

		GetUser,
		RegisterUser,
		GetMe,
		EditMe,
		ChangeMyIcon,
		ChangeMyPassword,
		EditOtherUsers,
		GetUserQRCode,
		GetUserGroup,
		CreateUserGroup,
		CreateSpecialUserGroup,
		EditUserGroup,
		DeleteUserGroup,

		GetUserTag,
		EditUserTag,

		WebRTC,

		GetClipFolder,
		CreateClipFolder,
		EditClipFolder,
		DeleteClipFolder,

		GetStampPalette,
		CreateStampPalette,
		EditStampPalette,
		DeleteStampPalette,
	}
	for _, p := range l {
		List.Add(p)
	}
}
