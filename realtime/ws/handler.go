package ws

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/traPtitech/traQ/realtime/viewer"
	"strings"
)

func (s *session) commandHandler(cmd string) {
	args := strings.Split(strings.TrimSpace(cmd), ":")

	switch strings.ToLower(args[0]) {
	case "viewstate":
		if len(args) < 2 {
			// 引数が不正
			s.sendErrorMessage(fmt.Sprintf("invalid args: %s", cmd))
			break
		}

		if strings.ToLower(args[1]) == "null" {
			s.streamer.realtime.ViewerManager.RemoveViewer(s)
			break
		}

		cid, err := uuid.FromString(args[1])
		if err != nil {
			// チャンネルIDが不正
			s.sendErrorMessage(fmt.Sprintf("invalid id: %s", args[1]))
			break
		}

		if len(args) < 3 {
			// 引数が不正
			s.sendErrorMessage(fmt.Sprintf("invalid args: %s", cmd))
			break
		}

		// TODO channelのアクセスチェック
		s.viewState.channelID = cid
		s.viewState.state = viewer.StateFromString(args[2])
		s.streamer.realtime.ViewerManager.SetViewer(s, s.userID, s.viewState.channelID, s.viewState.state)

	default:
		// 不明なコマンド
		s.sendErrorMessage(fmt.Sprintf("unknown command: %s", cmd))
	}
}

func (s *session) sendErrorMessage(error string) {
	_ = s.writeMessage(&rawMessage{
		t:    websocket.TextMessage,
		data: makeMessage("ERROR", error).toJSON(),
	})
}
