package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/penguinpowernz/nagent/hooks"
	"github.com/penguinpowernz/nagent/parsers/checkmk"
)

var ErrMergedShadowNotJSON = errors.New("merge shadows failed to produce valid JSON")

var (
	parserScriptHooks map[string]hooks.Modifier
	preScriptHooks    []hooks.Modifier
	sinkScriptHooks   []hooks.Sink
	postScriptHooks   []hooks.Modifier
)

func main() {
	var url, fn, home string
	flag.StringVar(&url, "u", os.Getenv("NATS_URL"), "the NATS server to connect to")
	flag.StringVar(&fn, "f", "", "agent report file to convert into JSON")
	flag.StringVar(&home, "d", "/var/lib/nagentd", "home dir for the program")
	flag.Parse()

	parserScriptHooks = hooks.BuildParsersFrom(home + "/parsers")
	preScriptHooks = hooks.BuildPreModifiersFrom(home + "/pre")
	sinkScriptHooks = hooks.BuildSinksFrom(home + "/sinks")
	postScriptHooks = hooks.BuildPostModifiersFrom(home + "/post")

	if fn != "" || fn == "-" {
		parseAndDumpFile(os.Stdin, nil)
		return
	}

	if fn != "" {
		parseAndDumpFile(os.Open(fn))
		return
	}

	if url == "" {
		url = nats.DefaultURL
	}

	nc, err := nats.Connect(url)
	if err != nil {
		panic(err)
	}

	store := &DiskStore{path: "./shadows"}

	nc.Subscribe("nagent.*.data", func(msg *nats.Msg) {
		if msg.Reply != "" {
			nc.Publish(msg.Reply, []byte("true"))
		}

		hostname := strings.Split(msg.Subject, ".")[1]

		// pass whole shadow through the pre-process rules
		preprocessed, err := hooks.ProcessPre(preScriptHooks, msg.Data)
		if err == hooks.ErrDiscardShadow {
			return // discard the shadow if necessary
		}

		// parse basic map string, with each section as an array of lines
		unparsed := checkmk.Parse(preprocessed)
		shadow := mkShadow(unparsed, parserScriptHooks)

		// save it to the store, which returns the JSON result of merging with the existing shadow
		data, err := store.Save(hostname, shadow)
		if err != nil {
			log.Println("ERROR: failed to save shadow")
			return
		}

		shadow, err = hooks.ProcessPost(postScriptHooks, data)
		switch err {
		case nil:
			if data, err = store.Save(hostname, shadow); err != nil {
				log.Println("ERROR: failed to save post scripts shadow")
				return
			}

		default:
			log.Println("ERROR: failed to modify shadow in post hooks:", err)
		}

		// if saving went OK pass it to all of our sinks
		if err == nil {
			hooks.ProcessSinks(sinkScriptHooks, hostname, data, keys(shadow))
		}
	})

	api := gin.Default()
	api.GET("/hosts", func(c *gin.Context) {
		c.JSON(200, store.Hosts())
	})

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

func mkShadow(unparsed map[string][]string, parserScripts map[string]hooks.Modifier) map[string]interface{} {
	shadow := map[string]interface{}{}

	// run through each section and give it to our filters to parse into JSON
	for section, lines := range unparsed {
		shadow[section] = lines // default is just an array of lines

		// this returns data, modified from the original raw lines
		data, err := hooks.ProcessParserSection(parserScripts, section, []byte(strings.Join(lines, "\n")))
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

	return shadow
}

func parseAndDumpFile(r io.Reader, err error) {
	if err != nil {
		panic(err)
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	// pass whole shadow through the pre-process rules
	preprocessed, err := hooks.ProcessPre(preScriptHooks, data)
	if err == hooks.ErrDiscardShadow {
		return // discard the shadow if necessary
	}

	// parse basic map string, with each section as an array of lines
	unparsed := checkmk.Parse(preprocessed)
	shadow := mkShadow(unparsed, parserScriptHooks)
	data, err = json.MarshalIndent(shadow, "", "  ")
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(data)
}

func keys(m map[string]interface{}) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}
