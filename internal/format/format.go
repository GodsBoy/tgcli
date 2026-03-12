package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Envelope wraps JSON output in a consistent structure.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *string     `json:"error,omitempty"`
}

// WriteJSON writes data as a JSON envelope to w.
func WriteJSON(w io.Writer, data interface{}) error {
	env := Envelope{Success: true, Data: data}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

// WriteError writes an error, either as JSON envelope or plain text to stderr.
func WriteError(w io.Writer, asJSON bool, err error) {
	if asJSON {
		msg := err.Error()
		env := Envelope{Success: false, Error: &msg}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(env)
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
