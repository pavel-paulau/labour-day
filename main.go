package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/hoisie/web"
)

var pckg_dir string

var data_source DataSource

func index() []byte {
	content, _ := ioutil.ReadFile(pckg_dir + "app/index.html")
	return content
}

type Config struct {
	CouchbaseAddress, ListenAddress, Release string
}

func main() {
	pckg_dir = os.Getenv("GOPATH") + "/src/github.com/pavel-paulau/labour-day/"
	web.Config.StaticDir = pckg_dir + "app"

	config_file, err := ioutil.ReadFile(pckg_dir + "config.json")
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	err = json.Unmarshal(config_file, &config)
	if err != nil {
		log.Fatal(err)
	}

	data_source = DataSource{config.CouchbaseAddress, config.Release}
	web.Get("/", index)
	web.Get("/timeline", data_source.GetTimeline)

	web.Run(config.ListenAddress)
}
