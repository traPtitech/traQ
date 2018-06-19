package model

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/oauth2"
	"github.com/traPtitech/traQ/oauth2/scope"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils/validator"
	"math"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// Bot Bot
type Bot interface {
	ID() uuid.UUID
	BotUserID() uuid.UUID
	Name() string
	DisplayName() string
	Description() string
	VerificationToken() string
	AccessTokenID() uuid.UUID
	PostURL() *url.URL
	SubscribeEvents() map[string]bool
	Activated() bool
	CreatorID() uuid.UUID
	InstallCode() string
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// GeneralBot Bot構造体
type GeneralBot struct {
	ID                string     `xorm:"char(36) not null pk"        validate:"uuid,required"`
	BotUserID         string     `xorm:"char(36) not null unique"    validate:"uuid,required"`
	Description       string     `xorm:"text not null"`
	VerificationToken string     `xorm:"text not null"               validate:"required"`
	AccessTokenID     string     `xorm:"char(36) not null"           validate:"uuid,required"`
	PostURL           string     `xorm:"text not null"               validate:"url,required"`
	SubscribeEvents   string     `xorm:"text not null"`
	Activated         bool       `xorm:"bool not null"`
	InstallCode       string     `xorm:"varchar(30) not null unique" validate:"required"`
	CreatorID         string     `xorm:"char(36) not null"           validate:"uuid,required"`
	CreatedAt         time.Time  `xorm:"created not null"`
	UpdatedAt         time.Time  `xorm:"updated not null"`
	DeletedAt         *time.Time `xorm:"timestamp"`
}

// TableName GeneralBotのテーブル名
func (*GeneralBot) TableName() string {
	return "bots"
}

// Validate 構造体を検証します
func (b *GeneralBot) Validate() error {
	return validator.ValidateStruct(b)
}

// GeneralBotUser GeneralBotUser構造体 内部にUser, GeneralBotを内包
type GeneralBotUser struct {
	*User       `xorm:"extends"`
	*GeneralBot `xorm:"extends"`
}

// TableName JOIN処理用
func (*GeneralBotUser) TableName() string {
	return "users"
}

// ID BotID
func (v *GeneralBotUser) ID() uuid.UUID {
	return uuid.Must(uuid.FromString(v.GeneralBot.ID))
}

// BotUserID BotのUserID
func (v *GeneralBotUser) BotUserID() uuid.UUID {
	return uuid.Must(uuid.FromString(v.GeneralBot.BotUserID))
}

// Name Bot名
func (v *GeneralBotUser) Name() string {
	return v.User.Name
}

// DisplayName Bot表示名
func (v *GeneralBotUser) DisplayName() string {
	return v.User.DisplayName
}

// Description Bot説明
func (v *GeneralBotUser) Description() string {
	return v.GeneralBot.Description
}

// VerificationToken Botの確認トークン
func (v *GeneralBotUser) VerificationToken() string {
	return v.GeneralBot.VerificationToken
}

// AccessTokenID BotのアクセストークンのID
func (v *GeneralBotUser) AccessTokenID() uuid.UUID {
	return uuid.Must(uuid.FromString(v.GeneralBot.AccessTokenID))
}

// PostURL BotのPOST URL
func (v *GeneralBotUser) PostURL() *url.URL {
	postURL, _ := url.Parse(v.GeneralBot.PostURL)
	return postURL
}

// SubscribeEvents Botの購読イベント
func (v *GeneralBotUser) SubscribeEvents() map[string]bool {
	return arrayToSet(strings.Fields(v.GeneralBot.SubscribeEvents))
}

// Activated Botが活性化されているか
func (v *GeneralBotUser) Activated() bool {
	return v.GeneralBot.Activated
}

// CreatorID Botの作成者ID
func (v *GeneralBotUser) CreatorID() uuid.UUID {
	return uuid.Must(uuid.FromString(v.GeneralBot.CreatorID))
}

// InstallCode Botのインストールコード
func (v *GeneralBotUser) InstallCode() string {
	return v.GeneralBot.InstallCode
}

// CreatedAt Botの作成日時
func (v *GeneralBotUser) CreatedAt() time.Time {
	return v.GeneralBot.CreatedAt
}

// UpdatedAt Botの更新日時
func (v *GeneralBotUser) UpdatedAt() time.Time {
	return v.GeneralBot.UpdatedAt
}

// CreateBot Botを作成します
func CreateBot(oauth2 *oauth2.Handler, name, displayName, description string, creatorID, iconFileID uuid.UUID, postURL *url.URL, subscribes []string) (Bot, error) {
	uid := uuid.NewV4()
	bid := uuid.NewV4()

	u := &User{
		ID:          uid.String(),
		Name:        fmt.Sprintf("BOT_%s", name),
		DisplayName: displayName,
		Email:       "",
		Password:    "",
		Salt:        "",
		Icon:        iconFileID.String(),
		Bot:         true,
		Role:        role.Bot.ID(),
	}
	if err := u.Validate(); err != nil {
		return nil, err
	}

	t, err := oauth2.IssueAccessToken(nil, uid, "", scope.AccessScopes{}, math.MaxInt32, false)
	if err != nil {
		return nil, err
	}

	gb := &GeneralBot{
		ID:                bid.String(),
		BotUserID:         uid.String(),
		Description:       description,
		VerificationToken: base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()),
		AccessTokenID:     t.ID.String(),
		PostURL:           postURL.String(),
		SubscribeEvents:   strings.Join(subscribes, " "),
		Activated:         false,
		CreatorID:         creatorID.String(),
		InstallCode:       base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()),
	}
	if err := gb.Validate(); err != nil {
		return nil, err
	}

	_, err = db.UseBool("activated", "bot").Insert(u, gb)
	if err != nil {
		return nil, err
	}
	return &GeneralBotUser{User: u, GeneralBot: gb}, nil
}

// UpdateBot Bot情報を更新します
func UpdateBot(id uuid.UUID, displayName, description *string, url *url.URL, subscribes []string) error {
	b, err := GetBot(id)
	if err != nil {
		return err
	}
	if b == nil {
		return ErrNotFound
	}

	if displayName != nil {
		if utf8.RuneCountInString(*displayName) > 64 {
			return errors.New("invalid displayName")
		}

		if _, err := db.ID(b.BotUserID().String()).Update(&User{
			DisplayName: *displayName,
		}); err != nil {
			return err
		}
	}

	if description != nil {
		if _, err := db.ID(b.ID().String()).Update(&GeneralBot{
			Description: *description,
		}); err != nil {
			return err
		}
	}

	if url != nil {
		if _, err := db.ID(b.ID().String()).UseBool("activated").Update(&GeneralBot{
			PostURL:   b.PostURL().String(),
			Activated: false,
		}); err != nil {
			return err
		}
	}

	if subscribes != nil {
		if _, err := db.ID(b.ID().String()).Update(&GeneralBot{
			SubscribeEvents: strings.Join(subscribes, " "),
		}); err != nil {
			return err
		}
	}

	return nil
}

// ActivateBot Botを活性化します
func ActivateBot(id uuid.UUID) error {
	b, err := GetBot(id)
	if err != nil {
		return err
	}
	if b == nil {
		return ErrNotFound
	}

	if _, err := db.ID(b.ID().String()).UseBool("activated").Update(&GeneralBot{
		Activated: true,
	}); err != nil {
		return err
	}
	return nil
}

// DeactivateBot Botを非活性化します
func DeactivateBot(id uuid.UUID) error {
	b, err := GetBot(id)
	if err != nil {
		return err
	}
	if b == nil {
		return ErrNotFound
	}

	if _, err := db.ID(b.ID().String()).UseBool("activated").Update(&GeneralBot{
		Activated: false,
	}); err != nil {
		return err
	}
	return nil
}

// ReissueBotTokens Botの現在のトークンを無効化し、新たに再発行します
func ReissueBotTokens(oauth2 *oauth2.Handler, id uuid.UUID) (Bot, string, error) {
	b, err := GetBot(id)
	if err != nil {
		return nil, "", err
	}
	if b == nil {
		return nil, "", ErrNotFound
	}

	if err := oauth2.DeleteTokenByID(b.AccessTokenID()); err != nil {
		return nil, "", err
	}

	t, err := oauth2.IssueAccessToken(nil, b.BotUserID(), "", scope.AccessScopes{}, math.MaxInt32, false)
	if err != nil {
		return nil, "", err
	}

	if _, err := db.ID(b.ID().String()).UseBool("activated").Update(&GeneralBot{
		VerificationToken: base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()),
		AccessTokenID:     t.ID.String(),
		Activated:         false,
	}); err != nil {
		return nil, "", err
	}

	b, err = GetBot(id)
	if err != nil {
		return nil, "", err
	}

	return b, t.AccessToken, nil
}

// DeleteBot Botを削除
func DeleteBot(id uuid.UUID) (err error) {
	_, err = db.Delete(&BotInstalledChannel{BotID: id.String()})
	if err != nil {
		return err
	}

	now := time.Now()
	_, err = db.Update(&GeneralBot{DeletedAt: &now}, &GeneralBot{ID: id.String()})
	return err
}

// GetBot Botを取得
func GetBot(id uuid.UUID) (Bot, error) {
	b := &GeneralBotUser{}
	if ok, err := db.Join("INNER", "bots", "bots.bot_user_id = users.id").Where("bots.id = ? AND bots.deleted_at IS NULL", id.String()).Get(&b); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b, nil
}

// GetBotsByCreator Botを取得
func GetBotsByCreator(id uuid.UUID) (arr []Bot, err error) {
	var gbs []*GeneralBotUser
	if err := db.Join("INNER", "bots", "bots.bot_user_id = users.id").Where("bots.creator_id = ? AND bots.deleted_at IS NULL", id.String()).Find(&gbs); err != nil {
		return nil, err
	}
	for _, v := range gbs {
		arr = append(arr, v)
	}
	return
}

// GetBotByInstallCode Botを取得
func GetBotByInstallCode(code string) (Bot, error) {
	b := &GeneralBotUser{}
	if ok, err := db.Join("INNER", "bots", "bots.bot_user_id = users.id").Where("bots.install_code = ? AND bots.deleted_at IS NULL", code).Get(&b); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b, nil
}

// GetAllBots Botを全て取得
func GetAllBots() (arr []Bot, err error) {
	var gbs []*GeneralBotUser
	if err := db.Join("INNER", "bots", "bots.bot_user_id = users.id").Where("bots.deleted_at IS NULL").Find(&gbs); err != nil {
		return nil, err
	}
	for _, v := range gbs {
		arr = append(arr, v)
	}
	return
}

// BotInstalledChannel BotInstalledChannel構造体
type BotInstalledChannel struct {
	BotID       string    `xorm:"char(36) not null unique(bot_channel)"`
	ChannelID   string    `xorm:"char(36) not null unique(bot_channel)"`
	InstalledBy string    `xorm:"char(36) not null"`
	CreatedAt   time.Time `xorm:"created not null"`
}

// TableName BotInstalledChannelのテーブル名
func (*BotInstalledChannel) TableName() string {
	return "bots_installed_channels"
}

// GetBID BotIDを取得
func (s *BotInstalledChannel) GetBID() uuid.UUID {
	return uuid.Must(uuid.FromString(s.BotID))
}

// GetCID ChannelIDを取得
func (s *BotInstalledChannel) GetCID() uuid.UUID {
	return uuid.Must(uuid.FromString(s.ChannelID))
}

// GetInstallerID InstallしたユーザーのIDを取得
func (s *BotInstalledChannel) GetInstallerID() uuid.UUID {
	return uuid.Must(uuid.FromString(s.InstalledBy))
}

// InstallBot Botをチャンネルにインストール
func InstallBot(botID, channelID, userID uuid.UUID) error {
	bic := &BotInstalledChannel{
		BotID:       botID.String(),
		ChannelID:   channelID.String(),
		InstalledBy: userID.String(),
	}

	_, err := db.InsertOne(bic)
	return err
}

// UninstallBot Botをチャンネルからアンインストール
func UninstallBot(botID, channelID uuid.UUID) error {
	bic := &BotInstalledChannel{
		BotID:     botID.String(),
		ChannelID: channelID.String(),
	}

	_, err := db.Delete(bic)
	return err
}

// GetBotInstalledChannels Botがインストールされているチャンネルを取得
func GetBotInstalledChannels(id uuid.UUID) (arr []BotInstalledChannel, err error) {
	err = db.Where("bot_id = ?", id.String()).Find(&arr)
	return
}

// GetInstalledBots チャンネルにインストールされているBotを取得
func GetInstalledBots(cid uuid.UUID) (arr []BotInstalledChannel, err error) {
	err = db.Where("channel_id = ?", cid.String()).Find(&arr)
	return
}

// BotOutgoingPostLog BotのPOST URLへのリクエストの結果のログ構造体
type BotOutgoingPostLog struct {
	RequestID  string `xorm:"char(36) not null pk"`
	BotID      string `xorm:"char(36) not null"`
	StatusCode int    `xorm:"int not null"`
	Request    string `xorm:"text not null"`
	Response   string `xorm:"text not null"`
	Error      string `xorm:"text not null"`
	Timestamp  int64  `xorm:"bigint not null"`
}

// TableName BotOutgoingPostLogのテーブル名
func (*BotOutgoingPostLog) TableName() string {
	return "bot_outgoing_post_logs"
}

// SavePostLog Botのポストログをdbに保存
func SavePostLog(reqID, botID uuid.UUID, status int, request, response, error string) error {
	l := &BotOutgoingPostLog{
		RequestID:  reqID.String(),
		BotID:      botID.String(),
		StatusCode: status,
		Request:    request,
		Response:   response,
		Error:      error,
		Timestamp:  time.Now().UnixNano(),
	}

	_, err := db.InsertOne(l)
	return err
}

func arrayToSet(arr []string) map[string]bool {
	s := map[string]bool{}
	for _, v := range arr {
		s[v] = true
	}
	return s
}
