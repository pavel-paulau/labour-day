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

func getTimeline() []byte {
	return data_source.GetTimeline(true)
}

func getRelTimeline() []byte {
	return data_source.GetTimeline(false)
}

func byPlatform(ctx *web.Context) []byte {
	build := ctx.Params["build"]
	return data_source.GetBreakdown(build, true)
}

func byPriority(ctx *web.Context) []byte {
	build := ctx.Params["build"]
	return data_source.GetBreakdown(build, false)
}

type Config struct {
	CouchbaseAddress, ListenAddress string
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

	data_source = DataSource{config.CouchbaseAddress}
	web.Get("/", index)
	web.Get("/abs_timeline", getTimeline)
	web.Get("/rel_timeline", getRelTimeline)
	web.Get("/by_priority", byPriority)
	web.Get("/by_platform", byPlatform)

	web.Run(config.ListenAddress)
}
