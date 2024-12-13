package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	Name string `json:"name"`
	Port int8 `json:"port"`
	Secret string `json:"secret"`
	Commands []Command `json:"commands"`
}

type Command struct {
	Shell string `json:"shell"`
	Target string `json:"target"`
}

func ReadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if (err != nil) {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if (err != nil) {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(bytes, &config)
	if (err != nil) {
		return nil, err
	}
	return &config, nil
}

func RunServer(config *Config) {
	writer := io.MultiWriter(
		os.Stdout,
	)

	http.HandleFunc("/run/", func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Path[1:] // Remove leading '/'
		for _, cmd := range config.Commands {
			if strings.EqualFold(cmd.Target, target) {
				err := executeCommand(cmd.Shell, writer)
				if (err != nil) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(err.Error()))
					return
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Command executed"))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Command not found"))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Printf("Starting server on port %d\n", config.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}

func executeCommand(shell string, writer io.Writer) (error) {
	cmd := exec.Command("sh", "-c", shell)
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}


func main() {
	config, err := ReadConfig("config.json")
	if (err != nil) {
		panic(err)
	}
	RunServer(config)
}