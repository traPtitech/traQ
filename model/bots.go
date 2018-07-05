package model

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
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
	GetID() uuid.UUID
	GetBotUserID() uuid.UUID
	GetName() string
	GetDisplayName() string
	GetDescription() string
	GetVerificationToken() string
	GetAccessTokenID() uuid.UUID
	GetPostURL() *url.URL
	GetSubscribeEvents() map[string]bool
	GetActivated() bool
	GetCreatorID() uuid.UUID
	GetInstallCode() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// GeneralBot Bot構造体
type GeneralBot struct {
	ID                string `gorm:"type:char(36);primary_key" validate:"uuid,required"`
	BotUserID         string `gorm:"type:char(36);unique"      validate:"uuid,required"`
	BotUser           User   `gorm:"foreignkey:BotUserID"`
	Description       string `gorm:"type:text"`
	VerificationToken string `gorm:"type:text"                 validate:"required"`
	AccessTokenID     string `gorm:"type:char(36)"             validate:"uuid,required"`
	PostURL           string `gorm:"type:text"                 validate:"url,required"`
	SubscribeEvents   string `gorm:"type:text"`
	Activated         bool
	InstallCode       string     `gorm:"type:varchar(30);unique"   validate:"required"`
	CreatorID         string     `gorm:"type:char(36)"             validate:"uuid,required"`
	CreatedAt         time.Time  `gorm:"precision:6"`
	UpdatedAt         time.Time  `gorm:"precision:6"`
	DeletedAt         *time.Time `gorm:"precision:6"`
}

// TableName GeneralBotのテーブル名
func (*GeneralBot) TableName() string {
	return "bots"
}

// Validate 構造体を検証します
func (b *GeneralBot) Validate() error {
	return validator.ValidateStruct(b)
}

// GetID BotのUUIDを返します
func (b *GeneralBot) GetID() uuid.UUID {
	return uuid.Must(uuid.FromString(b.ID))
}

// GetBotUserID BotのBotUserのUUIDを返します
func (b *GeneralBot) GetBotUserID() uuid.UUID {
	return uuid.Must(uuid.FromString(b.BotUserID))
}

// GetName Botの名前を返します
func (b *GeneralBot) GetName() string {
	return b.BotUser.Name
}

// GetDisplayName Botの表示名を返します
func (b *GeneralBot) GetDisplayName() string {
	return b.BotUser.DisplayName
}

// GetDescription Botの説明を返します
func (b *GeneralBot) GetDescription() string {
	return b.Description
}

// GetVerificationToken Botの認証トークンを返します
func (b *GeneralBot) GetVerificationToken() string {
	return b.VerificationToken
}

// GetAccessTokenID BotのアクセストークンのUUIDを返します
func (b *GeneralBot) GetAccessTokenID() uuid.UUID {
	return uuid.Must(uuid.FromString(b.AccessTokenID))
}

// GetPostURL BotのWebhookのURLを返します
func (b *GeneralBot) GetPostURL() *url.URL {
	postURL, _ := url.Parse(b.PostURL)
	return postURL
}

// GetSubscribeEvents Botの購読イベント一覧を返します
func (b *GeneralBot) GetSubscribeEvents() map[string]bool {
	return arrayToSet(strings.Fields(b.SubscribeEvents))
}

// GetActivated Botが活性化しているかどうかを返します
func (b *GeneralBot) GetActivated() bool {
	return b.Activated
}

// GetCreatorID Botの製作者のUUIDを返します
func (b *GeneralBot) GetCreatorID() uuid.UUID {
	return uuid.Must(uuid.FromString(b.CreatorID))
}

// GetInstallCode Botのインストールコードを返します
func (b *GeneralBot) GetInstallCode() string {
	return b.InstallCode
}

// GetCreatedAt Botの作成日時を返します
func (b *GeneralBot) GetCreatedAt() time.Time {
	return b.CreatedAt
}

// GetUpdatedAt Botの更新日時を返します
func (b *GeneralBot) GetUpdatedAt() time.Time {
	return b.UpdatedAt
}

// CreateBot Botを作成します
func CreateBot(oauth2 *oauth2.Handler, name, displayName, description string, creatorID, iconFileID uuid.UUID, postURL *url.URL, subscribes []string) (Bot, error) {
	uid := uuid.NewV4()
	bid := uuid.NewV4()

	u := &User{
		ID:          uid.String(),
		Name:        fmt.Sprintf("BOT_%s", name),
		DisplayName: displayName,
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

	tx := db.Begin()
	if err := tx.Error; err != nil {
		return nil, err
	}
	if err := tx.Create(u).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(gb).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	gb.BotUser = *u
	return gb, nil
}

// UpdateBot Bot情報を更新します
func UpdateBot(id uuid.UUID, displayName, description *string, url *url.URL, subscribes []string) error {
	b, err := getBot(id)
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

		if err := db.Model(&(b.BotUser)).Update("display_name", *displayName).Error; err != nil {
			return err
		}
	}

	if description != nil {
		if err := db.Model(b).Update("description", *description).Error; err != nil {
			return err
		}
	}

	if url != nil {
		if err := db.Model(b).Updates(map[string]interface{}{
			"post_url":  url.String(),
			"activated": false,
		}).Error; err != nil {
			return err
		}
	}

	if subscribes != nil {
		if err := db.Model(b).Update("subscribe_events", strings.Join(subscribes, " ")).Error; err != nil {
			return err
		}
	}

	return nil
}

// ActivateBot Botを活性化します
func ActivateBot(id uuid.UUID) error {
	b, err := getBot(id)
	if err != nil {
		return err
	}
	if b == nil {
		return ErrNotFound
	}

	return db.Model(b).Update("activated", true).Error
}

// DeactivateBot Botを非活性化します
func DeactivateBot(id uuid.UUID) error {
	b, err := getBot(id)
	if err != nil {
		return err
	}
	if b == nil {
		return ErrNotFound
	}

	return db.Model(b).Update("activated", false).Error
}

// ReissueBotTokens Botの現在のトークンを無効化し、新たに再発行します
func ReissueBotTokens(oauth2 *oauth2.Handler, id uuid.UUID) (Bot, string, error) {
	b, err := getBot(id)
	if err != nil {
		return nil, "", err
	}
	if b == nil {
		return nil, "", ErrNotFound
	}

	if err := oauth2.DeleteTokenByID(b.GetAccessTokenID()); err != nil {
		return nil, "", err
	}

	t, err := oauth2.IssueAccessToken(nil, b.GetBotUserID(), "", scope.AccessScopes{}, math.MaxInt32, false)
	if err != nil {
		return nil, "", err
	}

	err = db.Model(b).Updates(map[string]interface{}{
		"verification_token": base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()),
		"access_token_id":    t.ID.String(),
		"activated":          false,
	}).Error
	if err != nil {
		return nil, "", err
	}

	b, err = getBot(id)
	if err != nil {
		return nil, "", err
	}

	return b, t.AccessToken, nil
}

// DeleteBot Botを削除
func DeleteBot(id uuid.UUID) (err error) {
	err = db.Delete(BotInstalledChannel{}, BotInstalledChannel{BotID: id.String()}).Error
	if err != nil {
		return err
	}

	return db.Delete(GeneralBot{ID: id.String()}).Error
}

// GetBot Botを取得
func GetBot(id uuid.UUID) (Bot, error) {
	return getBot(id)
}

func getBot(id uuid.UUID) (*GeneralBot, error) {
	b := &GeneralBot{}
	if err := db.Preload("BotUser").Take(b, GeneralBot{ID: id.String()}).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// GetBotsByCreator Botを取得
func GetBotsByCreator(id uuid.UUID) (arr []Bot, err error) {
	var gbs []*GeneralBot
	if err := db.Preload("BotUser").Find(&gbs, GeneralBot{CreatorID: id.String()}).Error; err != nil {
		return nil, err
	}
	for _, v := range gbs {
		arr = append(arr, v)
	}
	return
}

// GetBotByInstallCode Botを取得
func GetBotByInstallCode(code string) (Bot, error) {
	b := &GeneralBot{}
	if err := db.Preload("BotUser").Take(b, GeneralBot{InstallCode: code}).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}

	return b, nil
}

// GetAllBots Botを全て取得
func GetAllBots() (arr []Bot, err error) {
	var gbs []*GeneralBot
	if err := db.Preload("BotUser").Find(&gbs).Error; err != nil {
		return nil, err
	}
	for _, v := range gbs {
		arr = append(arr, v)
	}
	return
}

// BotInstalledChannel BotInstalledChannel構造体
type BotInstalledChannel struct {
	BotID       string    `gorm:"type:char(36);unique_index:bot_channel"`
	ChannelID   string    `gorm:"type:char(36);unique_index:bot_channel"`
	InstalledBy string    `gorm:"type:char(36)"`
	CreatedAt   time.Time `gorm:"precision:6"`
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
	bic := BotInstalledChannel{
		BotID:       botID.String(),
		ChannelID:   channelID.String(),
		InstalledBy: userID.String(),
	}

	return db.Create(&bic).Error
}

// UninstallBot Botをチャンネルからアンインストール
func UninstallBot(botID, channelID uuid.UUID) error {
	target := BotInstalledChannel{
		BotID:     botID.String(),
		ChannelID: channelID.String(),
	}

	return db.Delete(BotInstalledChannel{}, target).Error
}

// GetBotInstalledChannels Botがインストールされているチャンネルを取得
func GetBotInstalledChannels(id uuid.UUID) (arr []BotInstalledChannel, err error) {
	err = db.Find(&arr, BotInstalledChannel{BotID: id.String()}).Error
	return
}

// GetInstalledBots チャンネルにインストールされているBotを取得
func GetInstalledBots(cid uuid.UUID) (arr []BotInstalledChannel, err error) {
	err = db.Find(&arr, BotInstalledChannel{ChannelID: cid.String()}).Error
	return
}

// BotOutgoingPostLog BotのPOST URLへのリクエストの結果のログ構造体
type BotOutgoingPostLog struct {
	RequestID  string `gorm:"type:char(36);primary_key"`
	BotID      string `gorm:"type:char(36)"`
	StatusCode int
	Request    string    `gorm:"type:text"`
	Response   string    `gorm:"type:text"`
	Error      string    `gorm:"type:text"`
	CreatedAt  time.Time `gorm:"precision:6"`
}

// TableName BotOutgoingPostLogのテーブル名
func (*BotOutgoingPostLog) TableName() string {
	return "bot_outgoing_post_logs"
}

// SavePostLog Botのポストログをdbに保存
func SavePostLog(reqID, botID uuid.UUID, status int, request, response, error string) error {
	l := BotOutgoingPostLog{
		RequestID:  reqID.String(),
		BotID:      botID.String(),
		StatusCode: status,
		Request:    request,
		Response:   response,
		Error:      error,
	}

	return db.Create(&l).Error
}

func arrayToSet(arr []string) map[string]bool {
	s := map[string]bool{}
	for _, v := range arr {
		s[v] = true
	}
	return s
}
