package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/penguinpowernz/nagent/hooks"
	"github.com/penguinpowernz/nagent/parsers/checkmk"
)

var ErrMergedShadowNotJSON = errors.New("merge shadows failed to produce valid JSON")

func main() {
	var url string
	flag.StringVar(&url, "u", os.Getenv("NATS_URL"), "the NATS server to connect to")
	flag.Parse()

	if url == "" {
		url = nats.DefaultURL
	}

	nc, err := nats.Connect(url)
	if err != nil {
		panic(err)
	}

	var sectionScriptHooks = hooks.BuildParsersFrom("./parsers")
	var ruleScriptHooks = hooks.BuildPreModifiersFrom("./pre")
	var sinkScriptHooks = hooks.BuildSinksFrom("./sinks")

	store := &DiskStore{path: "./shadows"}

	nc.Subscribe("nagent.*.data", func(msg *nats.Msg) {
		if msg.Reply != "" {
			nc.Publish(msg.Reply, []byte("true"))
		}

		// parse basic map string, with each section as an array of lines
		unparsed := checkmk.Parse(msg.Data)
		// pass whole shadow through the pre-process rules
		if err := hooks.ProcessPre(ruleScriptHooks, unparsed); err == hooks.ErrDiscardShadow {
			return // discard the shadow if necessary
		}

		hostname := strings.Split(msg.Subject, ".")[1]

		shadow := map[string]interface{}{}

		// run through each section and give it to our filters to parse into JSON
		for section, lines := range unparsed {
			shadow[section] = lines // default is just an array of lines

			// this returns data, modified from the original raw lines
			data, err := hooks.ProcessParsers(sectionScriptHooks, section, []byte(strings.Join(lines, "\n")))
			if err != nil {
				continue
			}

			var v interface{}
			if err := json.Unmarshal(data, &v); err != nil {
				continue // throw it away if it wasn't valid JSON
			}

			// replace the section with what was modified
			shadow[section] = v
		}

		// save it to the store, which returns the JSON result of merging with the existing shadow
		data, err := store.Save(hostname, shadow)
		if err != nil {
			log.Println("ERROR: failed to save shadow")
			return
		}

		// if saving went OK pass it to all of our sinks
		if err == nil {
			hooks.ProcessSinks(sinkScriptHooks, hostname, data)
		}
	})

	api := gin.Default()
	api.GET("/shadow/:name", func(c *gin.Context) {
		name := c.Param("name")
		data, err := store.Read(name)

		if err != nil {
			c.AbortWithStatus(404)
			return
		}

		c.Data(200, "application/json", data)
	})

	api.Run(":8080")
}
