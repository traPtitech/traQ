package messages

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	v1 "github.com/traPtitech/traQ/router/v4/gen/message/v1"
	"github.com/traPtitech/traQ/router/v4/gen/message/v1/v1connect"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
	mutil "github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/optional"
)

var _ v1connect.MessageServiceHandler = (*handler)(nil)

type handler struct {
	repo           repository.Repository
	channelManager channel.Manager
	messageManager message.Manager
	replacer       *mutil.Replacer
}

type Service struct {
	Path    string
	Handler http.Handler
}

func NewService(
	repo repository.Repository, cm channel.Manager,
	mm message.Manager, replacer *mutil.Replacer,
) *Service {
	h := &handler{
		repo:           repo,
		channelManager: cm,
		messageManager: mm,
		replacer:       replacer,
	}

	path, handler := v1connect.NewMessageServiceHandler(h)

	return &Service{Path: path, Handler: handler}
}

func (h *handler) ListChannelMessages(ctx context.Context, req *connect.Request[v1.ListChannelMessagesRequest]) (*connect.Response[v1.Messages], error) {
	// チャンネルIDの検証
	channelID, err := uuid.FromString(req.Msg.GetChannelId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// チャンネルの存在確認
	_, err = h.channelManager.GetChannel(channelID)
	if err != nil {
		switch err {
		case channel.ErrChannelNotFound:
			return nil, connect.NewError(connect.CodeNotFound, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	// クエリパラメータの処理
	query := message.TimelineQuery{
		Channel: channelID,
	}

	if req.Msg.Limit != nil {
		query.Limit = int(*req.Msg.Limit)
	}
	if req.Msg.Offset != nil {
		query.Offset = int(*req.Msg.Offset)
	}
	if req.Msg.Since != nil {
		query.Since = optional.From(req.Msg.Since.AsTime())
	}
	if req.Msg.Until != nil {
		query.Until = optional.From(req.Msg.Until.AsTime())
	}
	if req.Msg.Inclusive != nil {
		query.Inclusive = *req.Msg.Inclusive
	}
	if req.Msg.Order != nil && *req.Msg.Order == v1.MessageOrder_MESSAGE_ORDER_ASCENDING {
		query.Asc = true
	}

	// メッセージタイムライン取得
	timeline, err := h.messageManager.GetTimeline(query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// レスポンス作成
	messages := make([]*v1.Message, len(timeline.Records()))
	for i, m := range timeline.Records() {
		messages[i] = convertToProtoMessage(m)
	}

	response := &v1.Messages{
		Messages: messages,
	}

	return connect.NewResponse(response), nil
}

func convertToProtoMessage(m message.Message) *v1.Message {
	protoMsg := &v1.Message{
		Id:        m.GetID().String(),
		UserId:    m.GetUserID().String(),
		ChannelId: m.GetChannelID().String(),
		Content:   m.GetText(),
		CreatedAt: timestamppb.New(m.GetCreatedAt()),
		UpdatedAt: timestamppb.New(m.GetUpdatedAt()),
		Pinned:    m.GetPin() != nil,
	}

	// スタンプの変換
	stamps := m.GetStamps()
	protoStamps := make([]*v1.MessageStamp, len(stamps))
	for i, stamp := range stamps {
		protoStamps[i] = &v1.MessageStamp{
			UserId:    stamp.UserID.String(),
			StampId:   stamp.StampID.String(),
			Count:     int32(stamp.Count),
			CreatedAt: timestamppb.New(stamp.CreatedAt),
			UpdatedAt: timestamppb.New(stamp.UpdatedAt),
		}
	}
	protoMsg.Stamps = protoStamps

	return protoMsg
}
func (h *handler) PostMessage(ctx context.Context, req *connect.Request[v1.PostMessageRequest]) (*connect.Response[v1.Message], error) {
	// TODO: ユーザー認証の実装
	userID := uuid.Nil

	channelID, err := uuid.FromString(req.Msg.GetChannelId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	content := req.Msg.GetContent()
	if req.Msg.GetEmbed() {
		content = h.replacer.Replace(content)
	}

	m, err := h.messageManager.Create(channelID, userID, content)
	if err != nil {
		switch err {
		case message.ErrChannelArchived:
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("this channel has been archived"))
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(convertToProtoMessage(m)), nil
}
func (h *handler) DeleteMessage(ctx context.Context, req *connect.Request[v1.DeleteMessageRequest]) (*connect.Response[emptypb.Empty], error) {
	// TODO: ユーザー認証の実装
	// ConnectRPCではv3のようなmiddlewareによるユーザー認証が使えないため、
	// リクエストからユーザーIDを取得する仕組みを別途実装する必要があります
	// ここでは仮実装として空のUUIDを使用
	userID := uuid.Nil // 実際の実装では認証から取得

	// メッセージIDの検証
	messageID, err := uuid.FromString(req.Msg.GetMessageId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// メッセージの取得
	m, err := h.messageManager.Get(messageID)
	if err != nil {
		switch err {
		case message.ErrNotFound:
			return nil, connect.NewError(connect.CodeNotFound, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	// 削除権限の確認
	if muid := m.GetUserID(); muid != userID {
		mUser, err := h.repo.GetUser(muid, false)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		switch mUser.GetUserType() {
		case model.UserTypeHuman:
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you are not allowed to delete this message"))
		case model.UserTypeBot:
			// BOTのメッセージの削除権限の確認
			bot, err := h.repo.GetBotByBotUserID(mUser.GetID())
			if err != nil {
				switch err {
				case repository.ErrNotFound: // deleted bot
					return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you are not allowed to delete this message"))
				default:
					return nil, connect.NewError(connect.CodeInternal, err)
				}
			}

			if bot.CreatorID != userID {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you are not allowed to delete this message"))
			}
		case model.UserTypeWebhook:
			// Webhookのメッセージの削除権限の確認
			wh, err := h.repo.GetWebhookByBotUserID(mUser.GetID())
			if err != nil {
				switch err {
				case repository.ErrNotFound: // deleted webhook
					return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you are not allowed to delete this message"))
				default:
					return nil, connect.NewError(connect.CodeInternal, err)
				}
			}

			if wh.GetCreatorID() != userID {
				return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("you are not allowed to delete this message"))
			}
		}
	}

	// メッセージ削除
	if err := h.messageManager.Delete(m.GetID()); err != nil {
		switch err {
		case message.ErrChannelArchived:
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("the channel of this message has been archived"))
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}
