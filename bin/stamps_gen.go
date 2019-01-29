//+build tools

package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

const (
	/*
		twemoji Copyright 2018 Twitter, Inc and other contributors
		Graphics licensed under CC-BY 4.0: https://creativecommons.org/licenses/by/4.0/
	*/
	emojiZip     = "https://github.com/twitter/twemoji/archive/v11.1.0.zip"
	emojiExt     = "svg"
	emojiSrcDir  = "twemoji-11.1.0/2/svg/"
	emojiData    = "https://raw.githubusercontent.com/emojione/emojione/master/emoji.json"
	emojiDestDir = "../data/twemoji"
	stampsYAML   = "../data/stamps.yml"
)

type emoji struct {
	Name       string `json:"name"`
	Category   string `json:"category"`
	Order      int    `json:"order"`
	ShortName  string `json:"shortname"`
	CodePoints struct {
		DefaultMatches []string `json:"default_matches"`
	} `json:"code_points"`
}

func main() {
	log.Println("downloading twemoji...")

	twemojiZip, err := ioutil.TempFile("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(twemojiZip.Name())
	{
		res, err := http.Get(emojiZip)
		if err != nil {
			log.Fatal(err)
		}
		io.Copy(twemojiZip, res.Body)
		res.Body.Close()
		twemojiZip.Close()
	}

	log.Println("done")
	log.Println("downloading emoji data...")

	emojis := map[string]*emoji{}
	{
		res, err := http.Get(emojiData)
		if err != nil {
			log.Fatal(err)
		}
		temp := map[string]*emoji{}
		if err := json.NewDecoder(res.Body).Decode(&temp); err != nil {
			log.Fatal(err)
		}
		res.Body.Close()
		for _, v := range temp {
			for _, s := range v.CodePoints.DefaultMatches {
				emojis[s] = v
			}
		}
	}

	log.Println("done")
	log.Println("extracting emojis...")

	zipfile, err := zip.OpenReader(twemojiZip.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer zipfile.Close()

	if err := os.MkdirAll(emojiDestDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	yamlfile, err := os.OpenFile(stampsYAML, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer yamlfile.Close()

	yamlfile.WriteString("stamps:\n")
	for _, v := range zipfile.File {
		if strings.HasPrefix(v.Name, emojiSrcDir) && !v.FileInfo().IsDir() && strings.HasSuffix(path.Ext(v.Name), emojiExt) {
			_, filename := path.Split(v.Name)
			code := strings.TrimSuffix(filename, "."+emojiExt)

			emoji, ok := emojis[code]
			if !ok {
				emoji, ok = emojis["00"+code]
				if !ok {
					continue
				}
			}

			if emoji.Category == "modifier" {
				continue
			}
			if strings.HasSuffix(emoji.Name, "skin tone") {
				continue
			}

			f, err := v.Open()
			if err != nil {
				log.Fatal(err)
			}

			outPath := path.Join(emojiDestDir, filename)
			out, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, v.Mode())
			if err != nil {
				log.Fatal(err)
			}
			if _, err := io.Copy(out, f); err != nil {
				log.Fatal(err)
			}
			out.Close()
			f.Close()

			yamlfile.WriteString(fmt.Sprintf("  %s:\n    file: %s\n", strings.Trim(emoji.ShortName, ":"), outPath))
		}
	}

	log.Println("done")
}
