package permission

// Permission パーミッション
type Permission string

// Name パーミッション名
func (p Permission) Name() string {
	return string(p)
}

// Permissions パーミッションセット
type Permissions map[Permission]struct{}

// Add セットに権限を追加します
func (set Permissions) Add(p Permission) {
	set[p] = struct{}{}
}

// Remove セットから権限を削除します
func (set Permissions) Remove(p Permission) {
	delete(set, p)
}

// Contains セットに指定した権限が含まれているかどうか
func (set Permissions) Contains(p Permission) bool {
	_, ok := set[p]
	return ok
}

// Array セットの権限の配列を返します
func (set Permissions) Array() []Permission {
	result := make([]Permission, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	return result
}

func PermissionsFromArray(perms []Permission) Permissions {
	res := Permissions{}
	for _, perm := range perms {
		res.Add(perm)
	}
	return res
}

var List = []Permission{
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
	GetMyStampRecommendations,
	DeleteMyStamp,

	GetChannelStar,
	EditChannelStar,

	GetUnread,
	DeleteUnread,

	GetUser,
	RegisterUser,
	GetMe,
	GetOIDCUserInfo,
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
	AllUserGroupsAdmin,

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
