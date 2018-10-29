package main

import (
	"encoding/json"
	"os"
)

var (
	u    = make(users)
	sMap = servers{ServerMap: make(map[string]*server)}
)

func saveJSON(path string, data interface{}) error {
	f, err := os.OpenFile("json/"+path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Error("error saving", path, err)
		return err
	}

	if err = json.NewEncoder(f).Encode(data); err != nil {
		log.Error("error saving", path, err)
		return err
	}
	return nil
}

func loadJSON(path string, v interface{}) error {
	f, err := os.OpenFile("json/"+path, os.O_RDONLY, 0600)
	if err != nil {
		log.Error("error loading", path, err)
		return err
	}

	if err := json.NewDecoder(f).Decode(v); err != nil {
		log.Error("error loading", path, err)
		return err
	}
	return nil
}

func cleanup() {
	for _, f := range []func() error{saveConfig, saveQueue, saveServers, saveUsers} {
		if err := f(); err != nil {
			log.Error("error cleaning up files", err)
		}
	}
	log.Info("Done cleanup. Exiting.")
}

func loadConfig() error {
	return loadJSON("config.json", conf)
}

func saveConfig() error {
	return saveJSON("config.json", conf)
}

func loadServers() error {
	sMap = servers{ServerMap: make(map[string]*server)}
	return loadJSON("servers.json", &sMap)
}

func saveServers() error {
	return saveJSON("servers.json", &sMap)
}

func loadUsers() error {
	u = make(map[string]*user)
	return loadJSON("users.json", u)
}

func saveUsers() error {
	return saveJSON("users.json", u)
}

func loadQueue() error {
	return loadJSON("queue.json", imageQueue)
}

func saveQueue() error {
	return saveJSON("queue.json", imageQueue)
}
