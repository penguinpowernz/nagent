package main

import (
	"bufio"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	var key, fn, url string
	host, _ := os.Hostname()
	flag.StringVar(&key, "k", "", "the key to send data from")
	flag.StringVar(&fn, "f", "", "the file whose contents to send in the given key")
	flag.StringVar(&host, "host", host, "the hostname to use")
	flag.StringVar(&url, "u", nats.DefaultURL, "the hostname to use")
	flag.Parse()

	topic := "nagent." + host + ".data"

	nc, err := nats.Connect(url)
	if err != nil {
		panic(err)
	}

	// send all of the input under the given key
	if key != "" {
		src := io.Reader(os.Stdin) // default is stdin
		if fn != "-" && fn != "" { // but check if a file was provided, in order that it's contents be used
			f, err := os.Open(fn)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			src = f
		}

		// read from the source
		src = bufio.NewReader(src)
		data, err := ioutil.ReadAll(src)
		if err != nil {
			panic(err)
		}

		// add the section header on
		data = append([]byte(`<<<`+key+">>>\n"), data...)

		// send it off
		for {
			_, err = nc.Request(topic, data, time.Second*5)
			if err != nil {
				time.Sleep(time.Second * 2)
				continue
			}
			return
		}
	}

	// this is for the case where we have a saved checkmk style file that
	// is ready to be sent
	if fn != "" {
		src := io.Reader(os.Stdin) // use stdin if file was set as -
		if fn != "-" && fn != "" {
			f, err := os.Open(fn)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			src = f
		}

		src = bufio.NewReader(src)
		data, err := ioutil.ReadAll(src)
		if err != nil {
			panic(err)
		}

		send(nc, topic, data)
		return
	}

	// by default we just read from stdin
	src := io.Reader(os.Stdin)
	src = bufio.NewReader(src)
	data, err := ioutil.ReadAll(src)
	if err != nil {
		panic(err)
	}
	send(nc, topic, data)
}

func send(nc *nats.Conn, topic string, data []byte) (err error) {
	if err == nil {
		for {
			_, err := nc.Request(topic, data, 2*time.Second)
			if err == nil {
				break
			}
		}
	}

	return err
}
