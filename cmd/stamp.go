package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormzap"
	"github.com/traPtitech/traQ/utils/optional"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
)

const (
	/*
		twemoji Copyright 2019 Twitter, Inc and other contributors
		Graphics licensed under CC-BY 4.0: https://creativecommons.org/licenses/by/4.0/
	*/
	emojiZipURL  = "https://github.com/twitter/twemoji/archive/v12.1.5.zip"
	emojiDir     = "twemoji-12.1.5/assets/svg/"
	emojiMetaURL = "https://raw.githubusercontent.com/emojione/emojione/master/emoji.json"
)

type emojiMeta struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	Order      int    `json:"order"`
	ShortName  string `json:"shortname"`
	CodePoints struct {
		FullyQualified string   `json:"fully_qualified"`
		DefaultMatches []string `json:"default_matches"`
	} `json:"code_points"`
}

// stampCommand traQスタンプ操作コマンド
func stampCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "stamp",
		Short: "manage stamps",
	}

	cmd.AddCommand(
		stampInstallEmojisCommand(),
	)

	return &cmd
}

// stampInstallEmojisCommand ユニコード絵文字スタンプをインストールするコマンド
func stampInstallEmojisCommand() *cobra.Command {
	var update bool

	cmd := cobra.Command{
		Use:   "install-emojis",
		Short: "download and install Unicode emojiMeta stamps",
		Run: func(cmd *cobra.Command, args []string) {
			// Logger
			logger := getCLILogger()
			defer logger.Sync()

			// Database
			db, err := c.getDatabase()
			if err != nil {
				logger.Fatal("failed to connect database", zap.Error(err))
			}
			db.SetLogger(gormzap.New(logger.Named("gorm")))
			defer db.Close()

			// FileStorage
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}

			// Repository チャンネルツリー作ってないので注意
			repo, err := repository.NewGormRepository(db, fs, hub.New(), logger)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			if err := installEmojis(repo, logger, update); err != nil {
				logger.Fatal(err.Error())
			}

			logger.Info("done!")
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&update, "update", false, "update(replace) existing Unicode emojiMeta stamp's image files")

	return &cmd
}

func installEmojis(repo repository.Repository, logger *zap.Logger, update bool) error {
	// 絵文字メタデータをダウンロード
	logger.Info("downloading meta data...: " + emojiMetaURL)
	emojis, err := downloadEmojiMeta()
	if err != nil {
		return err
	}
	logger.Info("finished downloading meta data")

	// 絵文字画像データをダウンロード
	logger.Info("downloading twemoji...: " + emojiZipURL)
	twemojiZip, err := downloadEmojiZip()
	if err != nil {
		return err
	}
	logger.Info("finished downloading twemoji")

	// 絵文字解凍・インストール
	zipfile, err := zip.NewReader(twemojiZip, twemojiZip.Size())
	if err != nil {
		return err
	}

	saveEmojiFile := func(file *zip.File) (model.File, error) {
		_, filename := path.Split(file.Name)
		r, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()

		return repo.SaveFile(repository.SaveFileArgs{
			FileName: filename,
			FileSize: file.FileInfo().Size(),
			FileType: model.FileTypeStamp,
			Src:      r,
		})
	}

	logger.Info("installing emojis...")
	for _, file := range zipfile.File {
		if file.FileInfo().IsDir() {
			continue
		}

		dir, filename := path.Split(file.Name)
		if dir != emojiDir || !strings.HasSuffix(filename, ".svg") {
			continue
		}

		code := strings.TrimSuffix(filename, ".svg")
		emoji, ok := emojis[code]
		if !ok {
			emoji, ok = emojis["00"+code]
			if !ok {
				continue
			}
		}

		name := strings.Trim(emoji.ShortName, ":")
		s, err := repo.GetStampByName(name)
		if err != nil && err != repository.ErrNotFound {
			return err
		}

		if s == nil {
			// 新規追加
			meta, err := saveEmojiFile(file)
			if err != nil {
				return err
			}

			s, err := repo.CreateStamp(repository.CreateStampArgs{
				Name:      name,
				FileID:    meta.GetID(),
				CreatorID: uuid.Nil,
				IsUnicode: true,
			})
			if err != nil {
				return err
			}

			logger.Info(fmt.Sprintf("stamp added: %s (%s)", name, s.ID))
		} else {
			if !update {
				continue
			}

			// 既存のファイルを置き換え
			meta, err := saveEmojiFile(file)
			if err != nil {
				return err
			}

			if err := repo.UpdateStamp(s.ID, repository.UpdateStampArgs{
				FileID: optional.UUIDFrom(meta.GetID()),
			}); err != nil {
				return err
			}

			logger.Info(fmt.Sprintf("stamp updated: %s (%s)", name, s.ID))
		}
	}
	logger.Info("finished installing emojis")

	return nil
}

func downloadEmojiZip() (*bytes.Reader, error) {
	res, err := http.Get(emojiZipURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	return bytes.NewReader(b), err
}

func downloadEmojiMeta() (map[string]*emojiMeta, error) {
	res, err := http.Get(emojiMetaURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var temp map[string]*emojiMeta
	if err := jsoniter.ConfigFastest.NewDecoder(res.Body).Decode(&temp); err != nil {
		return nil, err
	}

	emojis := map[string]*emojiMeta{}
	for _, v := range temp {
		if v.Category == "modifier" {
			continue
		}
		if strings.HasSuffix(v.Name, "skin tone") {
			continue
		}

		emojis[v.CodePoints.FullyQualified] = v
		for _, s := range v.CodePoints.DefaultMatches {
			emojis[s] = v
		}
	}
	return emojis, nil
}
