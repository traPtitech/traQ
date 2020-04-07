package cmd

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path"
)

type dataRoot struct {
	Channels map[string]*dataChannel `yaml:"channels"`
	Stamps   map[string]*dataStamp   `yaml:"stamps"`
}

type dataStamp struct {
	File string `yaml:"file"`
}

func (d dataStamp) Validate() error {
	return vd.ValidateStruct(&d,
		vd.Field(&d.File, vd.Required),
	)
}

type dataChannel struct {
	Topic    string                  `yaml:"topic"`
	Force    bool                    `yaml:"force"`
	Children map[string]*dataChannel `yaml:"children"`
}

func insertInitialData(repo repository.Repository, initDataDir string, data *dataRoot) error {
	if err := createStamps(repo, initDataDir, data.Stamps); err != nil {
		return err
	}
	if err := createChannels(repo, data.Channels); err != nil {
		return err
	}
	return nil
}

func createStamps(repo repository.Repository, initDataDir string, stamps map[string]*dataStamp) error {
	for name, data := range stamps {
		if err := vd.Validate(name, validator.StampNameRuleRequired...); err != nil {
			return err
		}
		if err := data.Validate(); err != nil {
			return err
		}

		filepath := path.Join(initDataDir, data.File)
		_, filename := path.Split(filepath)
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		stat, _ := f.Stat()
		meta, err := repo.SaveFile(repository.SaveFileArgs{
			FileName: filename,
			FileSize: stat.Size(),
			FileType: model.FileTypeStamp,
			Src:      f,
		})
		f.Close()
		if err != nil {
			return err
		}
		if _, err := repo.CreateStamp(name, meta.GetID(), uuid.Nil); err != nil {
			return err
		}
	}
	return nil
}

func createChannels(repo repository.Repository, channels map[string]*dataChannel) error {
	for name, data := range channels {
		_, err := channelTreeTraverse(repo, name, data, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func channelTreeTraverse(repo repository.Repository, name string, node *dataChannel, parent *model.Channel) (*model.Channel, error) {
	if err := vd.Validate(name, validator.ChannelNameRuleRequired...); err != nil {
		return nil, err
	}

	parentID := uuid.Nil
	if parent != nil {
		parentID = parent.ID
	}
	ch, err := repo.CreatePublicChannel(name, parentID, uuid.Nil)
	if err != nil {
		return nil, err
	}

	if err := repo.UpdateChannel(ch.ID, repository.UpdateChannelArgs{
		Topic:              null.StringFrom(node.Topic),
		ForcedNotification: null.BoolFrom(node.Force),
	}); err != nil {
		return nil, err
	}

	for k, v := range node.Children {
		if v != nil {
			_, err := channelTreeTraverse(repo, k, v, ch)
			if err != nil {
				return nil, err
			}
		}
	}

	return ch, nil
}

func unmarshalInitData(r io.Reader) (*dataRoot, error) {
	data := dataRoot{}
	if err := yaml.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func initData(repo repository.Repository, initDataDir string) error {
	if stat, err := os.Stat(initDataDir); err != nil {
		return nil
	} else if !stat.IsDir() {
		return nil
	}

	files, err := ioutil.ReadDir(initDataDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Mode().IsRegular() && path.Ext(f.Name()) == ".yml" {
			is, err := os.Open(path.Join(initDataDir, f.Name()))
			if err != nil {
				return err
			}
			data, err := unmarshalInitData(is)
			is.Close()
			if err != nil {
				return err
			}

			if err := insertInitialData(repo, initDataDir, data); err != nil {
				return err
			}
		}
	}
	return nil
}
