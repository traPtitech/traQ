package ws

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

func (s *session) commandHandler(cmd string) {
	args := strings.Split(strings.TrimSpace(cmd), ":")

Command:
	switch strings.ToLower(args[0]) {
	case "rtcstate":
		// rtcstate:{チャンネルID}:({状態}:{セッションID})*
		if len(args) < 2 {
			// 引数が不正
			s.sendErrorMessage(fmt.Sprintf("invalid args: %s", cmd))
			break
		}

		// {チャンネルID} or null
		if str := strings.ToLower(args[1]); str == "null" || str == "" {
			// リセット
			if s.streamer.webrtc.ResetState(s.key, s.userID) != nil {
				// 別のコネクションでロック中
				s.sendErrorMessage("your webrtc state is locked by another ws connection")
			}
			break
		}
		cid, err := uuid.FromString(args[1])
		if err != nil {
			// チャンネルIDが不正
			s.sendErrorMessage(fmt.Sprintf("invalid id: %s", args[1]))
			break
		}

		// ({状態}:{セッションID})*
		if len(args) < 3 {
			// 引数が不正
			s.sendErrorMessage(fmt.Sprintf("invalid args: %s", cmd))
			break
		}
		if str := strings.ToLower(args[2]); str == "null" || str == "" {
			// リセット
			if s.streamer.webrtc.ResetState(s.key, s.userID) != nil {
				// 別のコネクションでロック中
				s.sendErrorMessage("your webrtc state is locked by another ws connection")
			}
			break
		}

		if (len(args)-2)%2 == 0 {
			// 状態+セッションのペアが出来ていない
			s.sendErrorMessage(fmt.Sprintf("invalid args: %s", cmd))
			break
		}

		sessions := map[string]string{}
		for i := 1; i < len(args)/2; i++ {
			state, session := args[2*i], args[2*i+1]
			if len(state) == 0 || len(session) == 0 {
				// 状態+セッションのペアが出来ていない
				s.sendErrorMessage(fmt.Sprintf("invalid args: %s", cmd))
				break Command
			}
			sessions[session] = state
		}

		_ = s.streamer.webrtc.SetState(s.key, s.userID, cid, sessions)

	default:
		// 不明なコマンド
		s.sendErrorMessage(fmt.Sprintf("unknown command: %s", cmd))
	}
}

func (s *session) sendErrorMessage(messeage string) {
	_ = s.WriteMessage(&rawMessage{
		t:    websocket.TextMessage,
		data: makeErrorMessage(messeage).toJSON(),
	})
}
