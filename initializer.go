package main

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path"
)

type dataRoot struct {
	Tags     map[string]*dataTag     `yaml:"tags"`
	Channels map[string]*dataChannel `yaml:"channels"`
	Stamps   map[string]*dataStamp   `yaml:"stamps"`
}

type dataTag struct {
	Restricted bool   `yaml:"restricted"`
	Type       string `yaml:"type"       validate:"max=30"`
}

type dataStamp struct {
	File string `yaml:"file" validate:"required"`
}

type dataChannel struct {
	Topic    string                  `yaml:"topic"`
	Force    bool                    `yaml:"force"`
	Children map[string]*dataChannel `yaml:"children"`
}

func insertInitialData(data *dataRoot) error {
	if err := createTags(data.Tags); err != nil {
		return err
	}
	if err := createStamps(data.Stamps); err != nil {
		return err
	}
	if err := createChannels(data.Channels); err != nil {
		return err
	}
	return nil
}

func createTags(tags map[string]*dataTag) error {
	for name, options := range tags {
		if err := validator.ValidateVar(name, "max=30"); err != nil {
			return err
		}
		if err := validator.ValidateStruct(options); err != nil {
			return err
		}
		if _, err := model.CreateTag(name, options.Restricted, options.Type); err != nil {
			return err
		}
	}
	return nil
}

func createStamps(stamps map[string]*dataStamp) error {
	for name, data := range stamps {
		if err := validator.ValidateVar(name, "name"); err != nil {
			return err
		}
		if err := validator.ValidateStruct(data); err != nil {
			return err
		}

		filepath := path.Join(config.InitDataDirectory, data.File)
		_, filename := path.Split(filepath)
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		stat, _ := f.Stat()
		id, err := model.SaveFile(filename, f, stat.Size(), "", model.FileTypeStamp)
		f.Close()
		if err != nil {
			return err
		}
		if _, err := model.CreateStamp(name, id, uuid.Nil); err != nil {
			return err
		}
	}
	return nil
}

func createChannels(channels map[string]*dataChannel) error {
	for name, data := range channels {
		_, err := channelTreeTraverse(name, data, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func channelTreeTraverse(name string, node *dataChannel, parent *model.Channel) (*model.Channel, error) {
	if err := validator.ValidateVar(name, "name"); err != nil {
		return nil, err
	}
	if err := validator.ValidateStruct(node); err != nil {
		return nil, err
	}

	parentID := ""
	if parent != nil {
		parentID = parent.ID.String()
	}
	ch, err := model.CreatePublicChannel(parentID, name, model.ServerUser().ID)
	if err != nil {
		return nil, err
	}

	if err := model.UpdateChannelTopic(ch.ID, node.Topic, model.ServerUser().ID); err != nil {
		return nil, err
	}
	if err := model.UpdateChannelFlag(ch.ID, nil, &node.Force, model.ServerUser().ID); err != nil {
		return nil, err
	}

	for k, v := range node.Children {
		if v != nil {
			_, err := channelTreeTraverse(k, v, ch)
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

func initData() error {
	if stat, err := os.Stat(config.InitDataDirectory); err != nil {
		return nil
	} else if !stat.IsDir() {
		return nil
	}

	files, err := ioutil.ReadDir(config.InitDataDirectory)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.Mode().IsRegular() && path.Ext(f.Name()) == ".yml" {
			is, err := os.Open(path.Join(config.InitDataDirectory, f.Name()))
			if err != nil {
				return err
			}
			data, err := unmarshalInitData(is)
			is.Close()
			if err != nil {
				return err
			}

			if err := insertInitialData(data); err != nil {
				return err
			}
		}
	}
	return nil
}
