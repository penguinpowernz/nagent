package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrNoParser      = errors.New("no known parser")
	ErrDiscardShadow = errors.New("discard shadow")
	ErrAborted       = errors.New("command indicated the output should be ignored")
)

type Modifier func(section string, data []byte) ([]byte, error)

func modiferCommand(cmdpath string) Modifier {
	return func(section string, data []byte) ([]byte, error) {
		cmd := exec.Command(cmdpath)

		log.Printf("[parser.%s] data %d bytes", section, len(data))

		if section != "" {
			cmd.Env = append(cmd.Env, "SECTION="+section)
		}

		buf := bytes.NewBufferString("")
		ebuf := bytes.NewBufferString("")
		cmd.Stdout = io.MultiWriter(buf, ebuf)
		cmd.Stderr = ebuf

		in, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("%s failed to connect stdin pipe %s", section, err)
		}

		err = cmd.Start()
		if err != nil {
			return ebuf.Bytes(), fmt.Errorf("%s failed to start command %s", section, err)
		}

		_, err = in.Write(data)
		if err != nil {
			return ebuf.Bytes(), fmt.Errorf("%s failed to write data to the pipe %s", section, err)
		}

		in.Close()

		err = cmd.Wait()
		out := buf.Bytes()

		if err != nil {
			err = fmt.Errorf("%s %s", section, err)
			out = ebuf.Bytes()
		}
		if cmd.ProcessState.ExitCode() == 21 {
			err = ErrAborted
		}
		if cmd.ProcessState.ExitCode() == 22 {
			err = ErrDiscardShadow
		}
		return out, err
	}
}

func ProcessParserSection(cmds map[string]Modifier, section string, data []byte) ([]byte, error) {
	cmd, found := cmds[section]
	if !found {
		cmd, found = cmds["_"]
		if !found {
			return nil, ErrNoParser
		}
	}

	out, err := cmd(section, data)
	// command worked?
	if err != nil {
		log.Println("ERROR:", err)
		for _, l := range strings.Split(string(out), "\n") {
			log.Printf("[parser.%s] %s", section, l)
		}
		return nil, err
	}

	if os.Getenv("DEBUG") != "" {
		log.Printf("DEBUG [parser.%s] ====> %s", section, string(out))
	}

	// is valid JSON?
	var test interface{}
	if err := json.Unmarshal(out, &test); err != nil {
		return nil, err
	}

	return out, nil
}

func BuildParsersFrom(dir string) map[string]Modifier {
	ls, _ := filepath.Glob(dir + "/*")
	cmds := map[string]Modifier{}

	for _, fn := range ls {
		if !isExecutable(fn) {
			continue
		}
		section := filepath.Base(fn)
		cmds[section] = modiferCommand(fn)
	}

	return cmds
}

func isExecutable(fn string) (x bool) {
	fi, err := os.Stat(fn)
	if err != nil {
		return
	}
	if fi.Mode().IsRegular() {
		x = fi.Mode().Perm()&0100 != 0
	}
	return
}
