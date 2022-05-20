package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/iancoleman/strcase"
)

var (
	baseURL = "https://api.github.com/repos/discord/discord-api-docs/contents/docs/"
	docsURL = "https://discord.com/developers/"
)

type FilePreview struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
	Path string `json:"path"`
}

type FileContents struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

func Request[T any](url string, token string) (T, error) {
	var target T
	var err error

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return target, err
	}

	if !(token == "") {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return target, err
	} else if resp.StatusCode >= 400 {
		return target, fmt.Errorf("http error: code %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&target)

	return target, err
}

func FetchMarkdown(cfg *Config) {
	dirs, err := Request[[]FilePreview](baseURL, cfg.Token)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup

Loop:
	for _, d := range dirs {
		if d.Type != "dir" {
			continue
		}
		for _, ignore := range cfg.IgnoreDirs {
			if strings.EqualFold(ignore, d.Name) {
				continue Loop
			}
		}
		filePreviews, err := Request[[]FilePreview](d.URL, cfg.Token)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range filePreviews {
			wg.Add(1)
			go func(p FilePreview) {
				defer wg.Done()

				if p.Type != "file" || path.Ext(p.Name) != ".md" {
					return
				}

				f, err := Request[FileContents](p.URL, cfg.Token)
				if err != nil {
					log.Fatal(err)
				} else if f.Encoding != "base64" {
					log.Fatalf("unknown file encoding: %s\n", f.Encoding)
				}

				fmt.Printf("fetched %s\n", p.Name)

				outputFile := cfg.OutDir + "/md/" + p.Name
				docLink := docsURL + path.Dir(p.Path) + "/" + strcase.ToKebab(strings.TrimSuffix(p.Name, ".md"))

				dat, err := base64.StdEncoding.DecodeString(f.Content)
				if err != nil {
					log.Fatal(err)
				} else if err := os.WriteFile(outputFile, append([]byte(docLink+"\n"), dat...), 0644); err != nil {
					log.Fatal(err)
				}
			}(p)
		}
	}

	wg.Wait()
}
