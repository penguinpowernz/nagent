package main

import (
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/penguinpowernz/nagent/hooks"
	"github.com/penguinpowernz/nagent/parsers/checkmk"
)

type Server struct {
	store *DiskStore
	nc    *nats.Conn
}

func (svr *Server) SetupRoutes(r gin.IRouter) {
	r.GET("/hosts", svr.listHosts)
	r.GET("/shadow/:name", svr.getHost)
}

func (svr *Server) listHosts(c *gin.Context) {
	c.JSON(200, svr.store.Hosts())
}

func (svr *Server) getHost(c *gin.Context) {
	name := c.Param("name")

	if cursor := c.Query("cursor"); cursor != "" {
		icursor, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			c.AbortWithStatus(400)
			return
		}

		t, err := svr.store.ModTime(name)
		if err != nil {
			c.AbortWithStatus(404)
			return
		}

		if icursor > t.Unix() {
			c.AbortWithStatus(304)
			return
		}
	}

	data, err := svr.store.Read(name)
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	c.Data(200, "application/json", data)
}

func (svr *Server) receiveEvent(msg *nats.Msg) {
	if msg.Reply != "" {
		svr.nc.Publish(msg.Reply, []byte("true"))
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
	data, err := svr.store.Save(hostname, shadow)
	if err != nil {
		log.Println("ERROR: failed to save shadow")
		return
	}

	shadow, err = hooks.ProcessPost(postScriptHooks, data)
	switch err {
	case nil:
		if data, err = svr.store.Save(hostname, shadow); err != nil {
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
}
