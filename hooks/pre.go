package hooks

import (
	"path/filepath"
	"sort"
)

// BuildPreModifiersFrom builds a list of pre-hooks from the given directory
func BuildPreModifiersFrom(dir string) []Modifier {
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

// ProcessPre processes the data passed in, and applies all of the pre-hooks to it
func ProcessPre(cmds []Modifier, data []byte) ([]byte, error) {
	for _, cmd := range cmds {
		_data, err := cmd("", data)

		// command said to discard?
		if err == ErrDiscardShadow { // exit status 2 means discard this entire shadow
			return nil, ErrDiscardShadow
		}

		// command worked?
		if err != nil { // any other exit status means discard this particular scripts results
			continue
		}

		data = _data
	}

	return data, nil
}
