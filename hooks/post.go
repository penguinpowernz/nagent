package hooks

import (
	"encoding/json"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

func BuildPostModifiersFrom(dir string) []Modifier {
	ls, _ := filepath.Glob(dir + "/*")
	cmds := []Modifier{}

	sort.Strings(ls)

	for _, fn := range ls {
		if !isExecutable(fn) {
			continue
		}
		cmds = append(cmds, modiferCommand(fn))
	}

	return cmds
}

func ProcessPost(cmds []Modifier, data []byte) (map[string]interface{}, error) {
	for _, cmd := range cmds {
		out, err := cmd("", data)

		// the shadow cannot be discarded at this point
		// if err == ErrDiscardShadow {}

		// command was aborted
		if err == ErrAborted {
			continue
		}

		// command worked?
		if err != nil {
			log.Println("ERROR:", err)
			for _, l := range strings.Split(string(out), "\n") {
				log.Printf("[post] %s", l)
			}
			continue
		}

		// is valid JSON?
		test := map[string]interface{}{}
		if err := json.Unmarshal(out, &test); err != nil {
			continue
		}

		data = out
	}

	out := map[string]interface{}{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}

	return out, nil
}
