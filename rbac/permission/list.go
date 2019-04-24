package permission

import "github.com/mikespook/gorbac"

// 全パーミッションのリスト。パーミッションを新たに定義した場合はここに必ず追加すること
var list = map[string]gorbac.Permission{
	CreateChannel.ID():       CreateChannel,
	GetChannel.ID():          GetChannel,
	EditChannel.ID():         EditChannel,
	DeleteChannel.ID():       DeleteChannel,
	ChangeParentChannel.ID(): ChangeParentChannel,

	GetTopic.ID():  GetTopic,
	EditTopic.ID(): EditTopic,

	GetMessage.ID():        GetMessage,
	PostMessage.ID():       PostMessage,
	EditMessage.ID():       EditMessage,
	DeleteMessage.ID():     DeleteMessage,
	ReportMessage.ID():     ReportMessage,
	GetMessageReports.ID(): GetMessageReports,

	GetPin.ID():    GetPin,
	CreatePin.ID(): CreatePin,
	DeletePin.ID(): DeletePin,

	GetNotificationStatus.ID():     GetNotificationStatus,
	ChangeNotificationStatus.ID():  ChangeNotificationStatus,
	ConnectNotificationStream.ID(): ConnectNotificationStream,
	RegisterDevice.ID():            RegisterDevice,

	GetUser.ID():          GetUser,
	GetMe.ID():            GetMe,
	RegisterUser.ID():     RegisterUser,
	EditMe.ID():           EditMe,
	ChangeMyIcon.ID():     ChangeMyIcon,
	ChangeMyPassword.ID(): ChangeMyPassword,
	EditOtherUsers.ID():   EditOtherUsers,

	GetMySessions.ID():    GetMySessions,
	DeleteMySessions.ID(): DeleteMySessions,

	GetClip.ID():          GetClip,
	CreateClip.ID():       CreateClip,
	DeleteClip.ID():       DeleteClip,
	GetClipFolder.ID():    GetClipFolder,
	CreateClipFolder.ID(): CreateClipFolder,
	PatchClipFolder.ID():  PatchClipFolder,
	DeleteClipFolder.ID(): DeleteClipFolder,

	GetStar.ID():    GetStar,
	CreateStar.ID(): CreateStar,
	DeleteStar.ID(): DeleteStar,

	GetChannelVisibility.ID():    GetChannelVisibility,
	ChangeChannelVisibility.ID(): ChangeChannelVisibility,

	GetUnread.ID():    GetUnread,
	DeleteUnread.ID(): DeleteUnread,

	GetMutedChannels.ID(): GetMutedChannels,
	MuteChannel.ID():      MuteChannel,
	UnmuteChannel.ID():    UnmuteChannel,

	GetTag.ID():                  GetTag,
	AddTag.ID():                  AddTag,
	RemoveTag.ID():               RemoveTag,
	ChangeTagLockState.ID():      ChangeTagLockState,
	OperateForRestrictedTag.ID(): OperateForRestrictedTag,
	EditTag.ID():                 EditTag,

	GetStamp.ID():                 GetStamp,
	CreateStamp.ID():              CreateStamp,
	EditStamp.ID():                EditStamp,
	EditStampName.ID():            EditStampName,
	EditStampCreatedByOthers.ID(): EditStampCreatedByOthers,
	DeleteStamp.ID():              DeleteStamp,
	GetMessageStamp.ID():          GetMessageStamp,
	AddMessageStamp.ID():          AddMessageStamp,
	RemoveMessageStamp.ID():       RemoveMessageStamp,
	GetMyStampHistory.ID():        GetMyStampHistory,

	UploadFile.ID():   UploadFile,
	DownloadFile.ID(): DownloadFile,
	DeleteFile.ID():   DeleteFile,

	GetHeartbeat.ID():  GetHeartbeat,
	PostHeartbeat.ID(): PostHeartbeat,

	GetWebhook.ID():          GetWebhook,
	CreateWebhook.ID():       CreateWebhook,
	EditWebhook.ID():         EditWebhook,
	DeleteWebhook.ID():       DeleteWebhook,
	AccessOthersWebhook.ID(): AccessOthersWebhook,

	GetBot.ID():          GetBot,
	CreateBot.ID():       CreateBot,
	EditBot.ID():         EditBot,
	DeleteBot.ID():       DeleteBot,
	ReissueBotToken.ID(): ReissueBotToken,
	InstallBot.ID():      InstallBot,
	UninstallBot.ID():    UninstallBot,

	GetMyTokens.ID():    GetMyTokens,
	RevokeMyToken.ID():  RevokeMyToken,
	GetClients.ID():     GetClients,
	CreateClient.ID():   CreateClient,
	EditMyClient.ID():   EditMyClient,
	DeleteMyClient.ID(): DeleteMyClient,

	GetMetrics.ID(): GetMetrics,
}

// GetPermission : パーミッション名からgorbac.Permissionを取得します
func GetPermission(name string) gorbac.Permission {
	return list[name]
}

// GetAllPermissionList : 全パーミッションリストを返します
func GetAllPermissionList() map[string]gorbac.Permission {
	return list
}
