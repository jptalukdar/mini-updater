package main

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type Config struct {
	Name       string    `json:"name"`
	Port       int16     `json:"port"`
	AuthScheme string    `json:"authScheme"` // BasicAuth or SecretKey
	Secret     string    `json:"secret"`
	Username   string    `json:"username"`
	Password   string    `json:"password"`
	Commands   []Command `json:"commands"`
}

type Command struct {
	Shell        string `json:"shell"`
	Target       string `json:"target"`
	StreamOutput bool   `json:"streamOutput"`
}

func ReadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func BasicAuthMiddleware(username, password string, w http.ResponseWriter, r *http.Request) bool {
	u, p, ok := r.BasicAuth()
	if ok {
		// Calculate SHA-256 hashes for the provided and expected
		// usernames and passwords.
		usernameHash := sha256.Sum256([]byte(u))
		passwordHash := sha256.Sum256([]byte(p))
		expectedUsernameHash := sha256.Sum256([]byte(username))
		expectedPasswordHash := sha256.Sum256([]byte(password))

		// Use the subtle.ConstantTimeCompare() function to check if
		// the provided username and password hashes equal the
		// expected username and password hashes. ConstantTimeCompare
		// will return 1 if the values are equal, or 0 otherwise.
		// Importantly, we should to do the work to evaluate both the
		// username and password before checking the return values to
		// avoid leaking information.
		usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
		passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

		// If the username and password are correct, then call
		// the next handler in the chain. Make sure to return
		// afterwards, so that none of the code below is run.
		if usernameMatch && passwordMatch {
			return true
		}
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	return false
}

func SecretKeyMiddleware(secretKey string, w http.ResponseWriter, r *http.Request) bool {

	if r.Header.Get("X-Secret-Key") != secretKey {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}

	return true

}

func HandleAuth(w http.ResponseWriter, r *http.Request, config *Config) bool {
	if strings.EqualFold(config.AuthScheme, "BasicAuth") {
		return BasicAuthMiddleware(config.Username, config.Password, w, r)
	} else if strings.EqualFold(config.AuthScheme, "SecretKey") {
		return SecretKeyMiddleware(config.Secret, w, r)
	}
	return true
}

func RunServer(config *Config, cs *CommandService) {
	// writer := io.MultiWriter(
	// 	os.Stdout,
	// )

	http.HandleFunc("/run/", func(w http.ResponseWriter, r *http.Request) {
		if !HandleAuth(w, r, config) {
			return
		}
		target := r.URL.Path[5:] // Remove leading '/'
		for _, cmd := range config.Commands {
			if strings.EqualFold(cmd.Target, target) {
				cs.SendCommand(cmd)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(
					fmt.Sprintf(`<html><body>Command submitted for execution. Check <a href="/status/%s">status</a> for output</body></html>`, target),
				))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Command not found"))
	})

	http.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		if !HandleAuth(w, r, config) {
			return
		}
		target := r.URL.Path[8:] // Remove leading '/'
		for _, cmd := range config.Commands {
			if strings.EqualFold(cmd.Target, target) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(cs.GetOutput(target)))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Command not found"))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !HandleAuth(w, r, config) {
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Printf("Starting server on port %d\n", config.Port)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}

func ExecuteCommand(shell string, writer io.Writer) error {
	cmd := exec.Command("sh", "-c", shell)
	cmd.Stdout = writer
	cmd.Stderr = writer
	return cmd.Run()
}

func main() {
	config, err := ReadConfig("config.json")
	if err != nil {
		panic(err)
	}

	cs := NewCommandService()
	cs.Start()

	// cs.SendCommand("command1")
	// cs.SendCommand("command2")

	// go func() {
	// 	for output := range cs.GetOutput() {
	// 		fmt.Println(output)
	// 	}
	// }()

	// time.Sleep(3 * time.Second) // Wait for all commands to be processed

	RunServer(config, cs)
}
