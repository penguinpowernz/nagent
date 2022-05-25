package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
)

var (
	ErrNoParser      = errors.New("no known parser")
	ErrDiscardShadow = errors.New("discard shadow")
)

type Modifier func(section string, data []byte) ([]byte, error)

func modiferCommand(cmdpath string) Modifier {
	return func(section string, data []byte) ([]byte, error) {
		cmd := exec.Command(cmdpath)

		if section != "" {
			cmd.Env = append(cmd.Env, "SECTION="+section)
		}

		buf := bytes.NewBufferString("")
		cmd.Stdout = buf

		in, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}

		err = cmd.Start()
		if err != nil {
			return nil, err
		}

		_, err = in.Write(data)
		if err != nil {
			return nil, err
		}

		in.Close()

		err = cmd.Wait()
		if cmd.ProcessState.ExitCode() == 2 {
			err = ErrDiscardShadow
		}
		return buf.Bytes(), err
	}
}

func ProcessParsers(cmds map[string]Modifier, section string, data []byte) ([]byte, error) {
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
		return nil, err
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
		section := filepath.Base(fn)
		cmds[section] = modiferCommand(fn)
	}

	return cmds
}
