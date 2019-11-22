package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/middlewares"
	v3middlewares "github.com/traPtitech/traQ/router/v3/middlewares"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type Handlers struct {
	RBAC     rbac.RBAC
	Repo     repository.Repository
	WS       *ws.Streamer
	Hub      *hub.Hub
	Logger   *zap.Logger
	Realtime *realtime.Service

	Version  string
	Revision string
}

// Setup APIルーティングを行います
func (h *Handlers) Setup(e *echo.Group) {
	// middleware preparation
	requires := middlewares.AccessControlMiddlewareGenerator(h.RBAC)
	retrieve := v3middlewares.NewParamRetriever(h.Repo)

	api := e.Group("/v3", middlewares.UserAuthenticate(h.Repo))
	{
		apiUsers := api.Group("/users")
		{
			apiUsers.GET("", NotImplemented)
			apiUsers.POST("", NotImplemented)
			apiUsersUid := apiUsers.Group("/:userID", retrieve.UserID(false))
			{
				apiUsersUid.GET("", NotImplemented)
				apiUsersUid.PATCH("", NotImplemented)
				apiUsersUid.POST("/messages", NotImplemented)
				apiUsersUid.GET("/messages", NotImplemented)
				apiUsersUid.GET("/icon", NotImplemented)
				apiUsersUid.PUT("/icon", NotImplemented)
				apiUsersUid.PUT("/password", NotImplemented)
				apiUsersUidTags := apiUsersUid.Group("/tags")
				{
					apiUsersUidTags.GET("", NotImplemented)
					apiUsersUidTags.POST("", NotImplemented)
					apiUsersUidTagsTid := apiUsersUidTags.Group("/:tagID")
					{
						apiUsersUidTagsTid.PATCH("", NotImplemented)
						apiUsersUidTagsTid.DELETE("", NotImplemented)
					}
				}
			}
			apiUsersMe := apiUsers.Group("/me")
			{
				apiUsersMe.GET("", NotImplemented)
				apiUsersMe.PATCH("", NotImplemented)
				apiUsersMe.GET("/stamp-history", NotImplemented)
				apiUsersMe.GET("/qr-code", h.GetMyQRCode)
				apiUsersMe.GET("/subscription", NotImplemented)
				apiUsersMe.PUT("/subscription/:channelID", NotImplemented)
				apiUsersMe.GET("/icon", NotImplemented)
				apiUsersMe.PUT("/icon", NotImplemented)
				apiUsersMe.PUT("/password", h.PutMyPassword)
				apiUsersMe.POST("/fcm-device", NotImplemented)
				apiUsersMeTags := apiUsersMe.Group("/tags")
				{
					apiUsersMeTags.GET("", NotImplemented)
					apiUsersMeTags.POST("", NotImplemented)
					apiUsersMeTagsTid := apiUsersMeTags.Group("/:tagID")
					{
						apiUsersMeTagsTid.PATCH("", NotImplemented)
						apiUsersMeTagsTid.DELETE("", NotImplemented)
					}
				}
				apiUsersMeStars := apiUsersMe.Group("/stars")
				{
					apiUsersMeStars.GET("", NotImplemented)
					apiUsersMeStars.POST("", NotImplemented)
					apiUsersMeStars.DELETE("/:channelID", NotImplemented)
				}
				apiUsersMe.GET("/unread", NotImplemented)
				apiUsersMe.DELETE("/unread", NotImplemented)
				apiUsersMe.GET("/sessions", NotImplemented)
				apiUsersMe.DELETE("/sessions/:sessionID", NotImplemented)
				apiUsersMe.GET("/tokens", NotImplemented)
				apiUsersMe.DELETE("/tokens/:tokenID", NotImplemented)
			}
		}
		apiChannels := api.Group("/channels")
		{
			apiChannels.GET("", NotImplemented)
			apiChannels.POST("", NotImplemented)
			apiChannelsCid := apiChannels.Group("/:channelID")
			{
				apiChannelsCid.GET("", NotImplemented)
				apiChannelsCid.PATCH("", NotImplemented)
				apiChannelsCid.GET("/messages", NotImplemented)
				apiChannelsCid.POST("/messages", NotImplemented)
				apiChannelsCid.GET("/stats", NotImplemented)
				apiChannelsCid.GET("/topic", NotImplemented)
				apiChannelsCid.PUT("/topic", NotImplemented)
				apiChannelsCid.GET("/viewers", NotImplemented)
				apiChannelsCid.GET("/pins", NotImplemented)
				apiChannelsCid.GET("/subscribers", NotImplemented)
				apiChannelsCid.PUT("/subscribers", NotImplemented)
				apiChannelsCid.PATCH("/subscribers", NotImplemented)
				apiChannelsCid.GET("/bots", NotImplemented)
				apiChannelsCid.GET("/events", NotImplemented)
			}
		}
		apiMessages := api.Group("/messages")
		{
			apiMessagesMid := apiMessages.Group("/:messageID")
			{
				apiMessagesMid.GET("", NotImplemented)
				apiMessagesMid.PUT("", NotImplemented)
				apiMessagesMid.DELETE("", NotImplemented)
				apiMessagesMid.GET("/pin", NotImplemented)
				apiMessagesMid.POST("/pin", NotImplemented)
				apiMessagesMid.DELETE("/pin", NotImplemented)
				apiMessagesMidStamps := apiMessagesMid.Group("/stamps")
				{
					apiMessagesMidStamps.GET("", NotImplemented)
					apiMessagesMidStampsSid := apiMessagesMidStamps.Group("/:stampID")
					{
						apiMessagesMidStampsSid.POST("", NotImplemented)
						apiMessagesMidStampsSid.DELETE("", NotImplemented)
					}
				}
			}
		}
		apiFiles := api.Group("/files")
		{
			apiFiles.GET("", NotImplemented)
			apiFiles.POST("", NotImplemented)
			apiFilesFid := apiFiles.Group("/:fileID")
			{
				apiFilesFid.GET("", NotImplemented)
				apiFilesFid.DELETE("", NotImplemented)
				apiFilesFid.GET("/meta", NotImplemented)
				apiFilesFid.GET("/thumbnail", NotImplemented)
			}
		}
		apiTags := api.Group("/tags")
		{
			apiTagsTid := apiTags.Group("/:tagID")
			{
				apiTagsTid.GET("", NotImplemented)
			}
		}
		apiStamps := api.Group("/stamps")
		{
			apiStamps.GET("", NotImplemented)
			apiStamps.POST("", NotImplemented)
			apiStampsSid := apiStamps.Group("/:stampID")
			{
				apiStampsSid.GET("", NotImplemented)
				apiStampsSid.DELETE("", NotImplemented)
			}
		}
		apiStampPalettes := api.Group("/stamp-palettes")
		{
			apiStampPalettes.GET("", NotImplemented)
			apiStampPalettes.POST("", NotImplemented)
			apiStampPalettesPid := apiStampPalettes.Group("/:paletteID")
			{
				apiStampPalettesPid.GET("", NotImplemented)
				apiStampPalettesPid.PATCH("", NotImplemented)
				apiStampPalettesPid.DELETE("", NotImplemented)
				apiStampPalettesPid.PUT("/stamps", NotImplemented)
			}
		}
		apiWebhooks := api.Group("/webhooks")
		{
			apiWebhooks.GET("", NotImplemented)
			apiWebhooks.POST("", NotImplemented)
			apiWebhooksWid := apiWebhooks.Group("/:webhookID")
			{
				apiWebhooksWid.GET("", NotImplemented)
				apiWebhooksWid.PATCH("", NotImplemented)
				apiWebhooksWid.DELETE("", NotImplemented)
				apiWebhooksWid.GET("/icon", NotImplemented)
				apiWebhooksWid.PUT("/icon", NotImplemented)
				apiWebhooksWid.GET("/messages", NotImplemented)
			}
		}
		apiGroups := api.Group("/groups")
		{
			apiGroups.GET("", NotImplemented)
			apiGroups.POST("", NotImplemented)
			apiGroupsGid := apiGroups.Group("/:groupID")
			{
				apiGroupsGid.GET("", NotImplemented)
				apiGroupsGid.PATCH("", NotImplemented)
				apiGroupsGid.DELETE("", NotImplemented)
				apiGroupsGidMembers := apiGroupsGid.Group("/members")
				{
					apiGroupsGidMembers.GET("", NotImplemented)
					apiGroupsGidMembers.POST("", NotImplemented)
					apiGroupsGidMembers.PUT("", NotImplemented)
					apiGroupsGidMembersUid := apiGroupsGidMembers.Group("/:userID")
					{
						apiGroupsGidMembersUid.PATCH("", NotImplemented)
						apiGroupsGidMembersUid.DELETE("", NotImplemented)
					}
				}
			}
		}
		apiActivity := api.Group("/activity")
		{
			apiActivity.GET("/timelines", NotImplemented)
			apiActivity.GET("/onlines", NotImplemented)
		}
		apiClients := api.Group("/clients")
		{
			apiClients.GET("", NotImplemented)
			apiClients.POST("", NotImplemented)
			apiClientsCid := apiClients.Group("/:clientID")
			{
				apiClientsCid.GET("", NotImplemented)
				apiClientsCid.PATCH("", NotImplemented)
				apiClientsCid.DELETE("", NotImplemented)
			}
		}
		apiBots := api.Group("/bots")
		{
			apiBots.GET("", NotImplemented)
			apiBots.POST("", NotImplemented)
			apiBotsBid := apiBots.Group("/:botID")
			{
				apiBotsBid.GET("", NotImplemented)
				apiBotsBid.PATCH("", NotImplemented)
				apiBotsBid.DELETE("", NotImplemented)
				apiBotsBid.GET("/icon", NotImplemented)
				apiBotsBid.PUT("/icon", NotImplemented)
				apiBotsBid.GET("/logs", NotImplemented)
				apiBotsBidActions := apiBotsBid.Group("/actions")
				{
					apiBotsBidActions.POST("/activate", NotImplemented)
					apiBotsBidActions.POST("/inactivate", NotImplemented)
					apiBotsBidActions.POST("/reissue", NotImplemented)
					apiBotsBidActions.POST("/join", NotImplemented)
					apiBotsBidActions.POST("/leave", NotImplemented)
				}
			}
		}
		apiWebRTC := api.Group("/webrtc")
		{
			apiWebRTC.GET("/state", NotImplemented)
			apiWebRTC.POST("/authenticate", NotImplemented)
		}
		apiClipFolders := api.Group("/clip-folders")
		{
			apiClipFolders.GET("", NotImplemented)
			apiClipFolders.POST("", NotImplemented)
			apiClipFoldersFid := apiClipFolders.Group("/:folderID")
			{
				apiClipFoldersFid.GET("", NotImplemented)
				apiClipFoldersFid.PATCH("", NotImplemented)
				apiClipFoldersFid.DELETE("", NotImplemented)
				apiClipFoldersFidMessages := apiClipFoldersFid.Group("/messages")
				{
					apiClipFoldersFidMessages.GET("", NotImplemented)
					apiClipFoldersFidMessages.POST("", NotImplemented)
					apiClipFoldersFidMessages.DELETE("/:messageID", NotImplemented)
				}
			}
		}
		api.GET("/ws", echo.WrapHandler(h.WS), requires(permission.ConnectNotificationStream))
	}

	apiNoAuth := e.Group("/v3")
	{
		apiNoAuth.GET("/version", h.GetVersion)
		apiNoAuth.POST("/login", NotImplemented)
		apiNoAuth.POST("/logout", NotImplemented)
		apiNoAuth.POST("/webhooks/:webhookID", NotImplemented)
		apiNoAuthPublic := apiNoAuth.Group("/public")
		{
			apiNoAuthPublic.GET("/icon/:username", NotImplemented)
		}
	}
}

func NotImplemented(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented)
}

func bindAndValidate(c echo.Context, i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}
	if err := vd.Validate(i); err != nil {
		if e, ok := err.(vd.InternalError); ok {
			return herror.InternalServerError(e.InternalError())
		}
		return herror.BadRequest(err)
	}
	return nil
}

func isTrue(s string) (b bool) {
	b, _ = strconv.ParseBool(s)
	return
}

func getRequestUser(c echo.Context) *model.User {
	return c.Get(consts.KeyUser).(*model.User)
}
