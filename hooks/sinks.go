package hooks

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Sink func(host string, data []byte, keys []string) ([]byte, error)

func BuildSinksFrom(dir string) []Sink {
	ls, _ := filepath.Glob(dir + "/*")
	cmds := []Sink{}

	sort.Strings(ls)

	for _, fn := range ls {
		if !isExecutable(fn) {
			continue
		}
		cmds = append(cmds, sinkCommand(fn))
	}

	return cmds
}

func sinkCommand(args ...string) Sink {
	return func(host string, data []byte, keys []string) ([]byte, error) {
		cmd := exec.Command(args[0])
		if len(args) > 1 {
			cmd.Args = append(cmd.Args, args[1:]...)
		}

		cmd.Env = append(cmd.Env, "HOSTNAME="+host)
		cmd.Env = append(cmd.Env, "CHANGED="+strings.Join(keys, ","))

		buf := bytes.NewBufferString("")
		cmd.Stderr = buf
		cmd.Stdout = buf

		in, err := cmd.StdinPipe()
		if err != nil {
			return buf.Bytes(), fmt.Errorf("%s: %s", args[0], err)
		}

		err = cmd.Start()
		if err != nil {
			return buf.Bytes(), fmt.Errorf("%s: %s", args[0], err)
		}

		_, err = in.Write(data)
		if err != nil {
			return buf.Bytes(), fmt.Errorf("%s: %s", args[0], err)
		}

		in.Close()
		if err := cmd.Wait(); err != nil {
			return buf.Bytes(), fmt.Errorf("%s: %s", args[0], err)
		}

		return nil, nil
	}
}

func ProcessSinks(cmds []Sink, host string, data []byte, keys []string) {
	for i, cmd := range cmds {
		go func(i int, cmd Sink) {
			output, err := cmd(host, data, keys)
			if err != nil {
				log.Println("ERROR: running sink", i, "failed:", err)
				log.Println(string(output))
			}
		}(i, cmd)
	}
}
