package types

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/robfig/cron"
)

type Manifest struct {
	Name           string            `json:"name,omitempty"`            // information field
	Description    string            `json:"description,omitempty"`     // information field
	Run            []string          `json:"run"`                       // command to run
	OutputHeaders  map[string]string `json:"output_headers,omitempty"`  // output headers
	InputHeaders   map[string]string `json:"input_headers,omitempty"`   // headers to map from request to environment
	Query          map[string]string `json:"query,omitempty"`           // map query or form parameters to environment
	Environment    map[string]string `json:"environment,omitempty"`     // custom environment
	Method         string            `json:"method,omitempty"`          // restrict invoke only to the HTTP method
	MethodEnv      string            `json:"method_env,omitempty"`      // map method name to environment
	PathEnv        string            `json:"path_env,omitempty"`        // map requested path to environment
	TimeLimit      JsonDuration      `json:"time_limit,omitempty"`      // time limit to run (zero is infinity)
	MaximumPayload int64             `json:"maximum_payload,omitempty"` // limit incoming payload (zero is unlimited)
	Cron           []Schedule        `json:"cron,omitempty"`            // crontab expression and action name to invoke
	Serial         bool              `json:"serial,omitempty"`          // serial execution - only one running instance of lambda allowed
	Static         string            `json:"static,omitempty"`          // relative path to static folder
}

type Schedule struct {
	Cron      string       `json:"cron"`       // crontab expression
	Action    string       `json:"action"`     // action to invoke
	TimeLimit JsonDuration `json:"time_limit"` // time limit to execute
}

func (mf *Manifest) Validate() error {
	for _, entry := range mf.Cron {
		if _, err := cron.Parse(entry.Cron); err != nil {
			return fmt.Errorf("bad cront expression for action %s (%s): %w", entry.Action, entry.Cron, err)
		}
	}
	return nil
}

func (mf *Manifest) SaveAs(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(mf)
}

func (mf *Manifest) LoadFrom(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(mf)
}

type JsonDuration time.Duration

func (j *JsonDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(*j).String())
}

func (j *JsonDuration) UnmarshalJSON(bytes []byte) error {
	var str string
	err := json.Unmarshal(bytes, &str)
	if err != nil {
		return err
	}
	v, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*j = JsonDuration(v)
	return nil
}
