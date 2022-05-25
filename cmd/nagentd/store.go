package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os/exec"
)

type DiskStore struct {
	path string
}

func (store *DiskStore) Read(host string) ([]byte, error) {
	return ioutil.ReadFile(store.path + "/" + host + ".json")
}

func (store *DiskStore) Save(host string, shadow map[string]interface{}) (data []byte, err error) {
	// find the existing shadow
	oldfn := store.path + "/" + host + ".json"
	if _, err = store.Read(host); err != nil {
		if err = ioutil.WriteFile(oldfn, []byte(`{}`), 0644); err != nil {
			return
		}
	}

	// save the incoming shadow
	newfn := "/tmp/" + host + ".json"
	data, err = json.Marshal(shadow)
	if err != nil {
		return
	}
	if err = ioutil.WriteFile(newfn, data, 0644); err != nil {
		return
	}

	// merge the shadows
	data, err = mergeShadow(oldfn, newfn)
	if err != nil {
		return
	}

	// save the merged shadow
	if err = ioutil.WriteFile(oldfn, data, 0644); err != nil {
		return
	}

	// return the merged shadow
	var v interface{}
	if err = json.Unmarshal(data, &v); err != nil {
		err = ErrMergedShadowNotJSON
	}

	return
}

func mergeShadow(old, incoming string) ([]byte, error) {
	stde := bytes.NewBufferString("")
	cmd := exec.Command("bash", "-c", "jq -s '.[0] * .[1]' "+old+" "+incoming)
	cmd.Stderr = stde
	data, err := cmd.Output()
	if err != nil {
		log.Printf("failed to run jq: %s: %s", stde, err)
	}
	return data, err
}
