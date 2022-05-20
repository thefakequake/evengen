package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
)

var (
	fetch     bool
	configFile string
)

func init() {
	flag.BoolVar(&fetch, "f", false, "fetch new markdown files from github")
	flag.StringVar(&configFile, "c", "config.json", "config file")
	flag.Parse()
}

func main() {
	var err error
	
	cfg, err := GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("loaded %s\n", configFile)


	for _, dir := range []string{cfg.OutDir, cfg.OutDir+"/go"} {
		if _, err := os.Stat(dir); err != nil {
			if !os.IsNotExist(err) {
				log.Fatal(err)
			}

			os.Mkdir(dir, 0)
			fetch = true
		}
	}

	if fetch {
		mdDir := cfg.OutDir+"/md"
		os.RemoveAll(mdDir)
		os.Mkdir(mdDir, 0)

		fmt.Println("fetching new markdown files...")
		FetchMarkdown(cfg)
	}

    dir, err := os.ReadDir(cfg.OutDir+"/go")
	if err != nil {
		log.Fatal(err)
	}
    for _, d := range dir {
        os.RemoveAll(path.Join([]string{cfg.OutDir+"/go", d.Name()}...))
    }

	fmt.Println("parsing markdown...")
	ParseMarkdown(cfg)
}