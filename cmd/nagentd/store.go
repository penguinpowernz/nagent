package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type DiskStore struct {
	path string
}

func (store *DiskStore) Read(host string) ([]byte, error) {
	return ioutil.ReadFile(store.path + "/" + host + ".json")
}

func (store *DiskStore) ModTime(host string) (time.Time, error) {
	stat, err := os.Stat(store.path + "/" + host + ".json")
	if err != nil {
		return time.Time{}, err
	}
	return stat.ModTime(), err
}

func (store *DiskStore) Hosts() []Host {
	files, _ := filepath.Glob(store.path + "/*.json")
	hosts := []Host{}
	for _, f := range files {
		hosts = append(hosts, NewHost(f))
	}

	return hosts
}

type Host struct {
	Name     string
	Filename string
	ModTime  time.Time
}

func NewHost(fn string) (h Host) {
	h.Filename = fn
	h.Name = strings.TrimSuffix(filepath.Base(fn), ".json")
	stat, err := os.Stat(fn)
	if err != nil {
		return
	}
	h.ModTime = stat.ModTime()
	return
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
	data, err = jqMergeJSONFiles(oldfn, newfn)
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

func jqMergeJSONFiles(old, incoming string) ([]byte, error) {
	stde := bytes.NewBufferString("")
	jqcmd := fmt.Sprintf("jq -s '.[0] * .[1]' %s %s", old, incoming)
	cmd := exec.Command("bash", "-c", jqcmd)
	cmd.Stderr = stde
	data, err := cmd.Output()
	if err != nil {
		log.Printf("failed to run jq: %s: %s", stde, err)
	}
	return data, err
}
