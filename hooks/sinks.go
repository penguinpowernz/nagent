package hooks

import (
	"log"
	"os/exec"
	"path/filepath"
	"sort"
)

type Sink func(host string, data []byte) error

func BuildSinksFrom(dir string) []Sink {
	ls, _ := filepath.Glob(dir + "/*")
	cmds := []Sink{}

	sort.Strings(ls)

	for _, fn := range ls {
		cmds = append(cmds, sinkCommand(fn))
	}

	return cmds
}

func sinkCommand(args ...string) Sink {
	return func(host string, data []byte) error {
		cmd := exec.Command(args[0])
		if len(args) > 1 {
			cmd.Args = append(cmd.Args, args[1:]...)
		}

		cmd.Env = append(cmd.Env, "HOSTNAME="+host)

		in, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		err = cmd.Start()
		if err != nil {
			return err
		}

		_, err = in.Write(data)
		if err != nil {
			return err
		}

		in.Close()
		return cmd.Wait()
	}
}

func ProcessSinks(cmds []Sink, host string, data []byte) {
	for i, cmd := range cmds {
		go func(i int, cmd Sink) {
			err := cmd(host, data)
			if err != nil {
				log.Println("ERROR: running sink", i, "failed:", err)
			}
		}(i, cmd)
	}
}
