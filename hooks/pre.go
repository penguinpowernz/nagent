package hooks

import (
	"encoding/json"
	"path/filepath"
	"sort"
)

func BuildPreModifiersFrom(dir string) []Modifier {
	ls, _ := filepath.Glob(dir + "/*")
	cmds := []Modifier{}

	sort.Strings(ls)

	for _, fn := range ls {
		cmds = append(cmds, modiferCommand(fn))
	}

	return cmds
}

func ProcessPre(cmds []Modifier, unparsed map[string][]string) error {
	data, _ := json.Marshal(unparsed)

	for _, cmd := range cmds {
		out, err := cmd("", data)

		// command said to discard?
		if err == ErrDiscardShadow {
			return ErrDiscardShadow
		}

		// command worked?
		if err != nil {
			continue
		}

		// is valid JSON?
		_t := map[string][]string{}
		if err := json.Unmarshal(out, &_t); err != nil {
			continue
		}

		data = out
	}

	return nil
}
