package twemoji

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/utils/optional"
)

const (
	/*
		twemoji Copyright 2019 Twitter, Inc and other contributors
		Graphics licensed under CC-BY 4.0: https://creativecommons.org/licenses/by/4.0/
	*/
	emojiZipURL  = "https://github.com/jdecked/twemoji/archive/refs/tags/v15.1.0.zip"
	emojiDir     = "twemoji-15.1.0/assets/svg/"
	emojiMetaURL = "https://raw.githubusercontent.com/joypixels/emoji-assets/v9.0.0/emoji.json"
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

var replaceNameMap = map[string]string{
	// 英数字以外の文字が含まれているので置き換え
	"pi\u00f1ata": "pinata",
	// 長すぎるので置き換え
	"face_with_open_eyes_and_hand_over_mouth":  "face_with_open_eyes_hand",
	"hand_with_index_finger_and_thumb_crossed": "hand_index_finger_thumb_crossed",
	"man_in_motorized_wheelchair_facing_right": "man_powered_wheelchair_right",
	"man_in_manual_wheelchair_facing_right":"man_manual_wheelchair_right",
	"woman_in_manual_wheelchair_facing_right":"woman_manual_wheelchair_right",
	"woman_in_motorized_wheelchair_facing_right":"woman_powered_wheelchair_right",
	"person_in_motorized_wheelchair_facing_right":"person_powered_wheelchair_right",
	"person_in_manual_wheelchair_facing_right":"woman_manual_wheelchair_right",
	"woman_with_white_cane_facing_right":"woman_white_cane_facing_right",
	"person_with_white_cane_facing_right":"person_white_cane_facing_right",
}

func Install(repo repository.Repository, fm file.Manager, logger *zap.Logger, update bool) error {
	logger = logger.Named("twemoji_installer")

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
	zipFile, err := zip.NewReader(twemojiZip, twemojiZip.Size())
	if err != nil {
		return err
	}

	saveEmojiFile := func(f *zip.File) (model.File, error) {
		_, filename := path.Split(f.Name)
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()

		return fm.Save(file.SaveArgs{
			FileName: filename,
			FileSize: f.FileInfo().Size(),
			FileType: model.FileTypeStamp,
			Src:      r,
		})
	}

	logger.Info("installing emojis...")
	for _, file := range zipFile.File {
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
		if replacedName, ok := replaceNameMap[name]; ok {
			name = replacedName
		}

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
				return fmt.Errorf("failed to create stamp (name: %s): %w", name, err)
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
				FileID: optional.From(meta.GetID()),
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

	b, err := io.ReadAll(res.Body)
	return bytes.NewReader(b), err
}

func downloadEmojiMeta() (map[string]*emojiMeta, error) {
	res, err := http.Get(emojiMetaURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var temp map[string]*emojiMeta
	if err := jsonIter.ConfigFastest.NewDecoder(res.Body).Decode(&temp); err != nil {
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
