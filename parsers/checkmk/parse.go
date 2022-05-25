package checkmk

import "strings"

func Parse(data []byte) (doc map[string][]string) {
	doc = map[string][]string{}

	var currsec string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "<<<") {
			currsec = strings.Trim(line, "<>")
			continue
		}

		doc[currsec] = append(doc[currsec], line)
	}

	return
}

// func DeepParse(data []byte) (out map[string]interface{}) {
// 	doc := map[string]interface{}{}

// 	for section, lines := range Parse(data) {
// 		switch section {
// 		case "check_mk":

// 		case ""

// 		default:
// 			doc[section] = lines
// 		}
// 	}
// }
