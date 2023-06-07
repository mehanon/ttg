package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"ttg2/lib"
)

const DefaultConfigPath = "./cfg.json"

func main() {
	configPath := flag.String("cfg", DefaultConfigPath,
		"path to the config file (may be useful to run multiple bots in parallel)")
	debug := flag.Bool("debug", false, "print debug log")
	flag.Parse()

	lib.Debug = *debug

	buffer, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalln(err)
	}
	var cfg lib.TtTgConfig
	err = json.Unmarshal(buffer, &cfg)
	if err != nil {
		log.Fatalln(err)
	}

	tttg, err := lib.NewTtTg(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	tttg.Start()
}
