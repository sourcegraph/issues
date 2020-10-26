package store

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type DockerStep struct {
	Root     string   `json:"root"`
	Image    string   `json:"image"`
	Commands []string `json:"commands"`
}

func (n *DockerStep) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("value is not []byte: %T", value)
	}

	return json.Unmarshal(b, &n)
}

func (n DockerStep) Value() (driver.Value, error) {
	return json.Marshal(n)
}
