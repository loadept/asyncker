package cmd

import (
	"encoding/gob"
	"encoding/json/v2"
	"fmt"
	"os"
	"strings"
)

type Task struct {
	ID         int
	Name       string
	PID        int
	CmdLine    CmdLine
	ExecutedAt int64
	Logs       Logs
}

type CmdLine struct {
	Command string
	Args    []string
}

func (c CmdLine) MarshalJSON() ([]byte, error) {
	return json.Marshal(fmt.Sprintf("%s %s", c.Command, strings.Join(c.Args, " ")))
}

type Logs struct {
	Output string
	Error  string
}

func LoadTasks(filePath string) ([]Task, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	defer file.Close()

	if stat, err := file.Stat(); err != nil {
		return nil, err
	} else if stat.Size() == 0 {
		return []Task{}, nil
	}

	var tasks []Task
	if err := gob.NewDecoder(file).Decode(&tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func SaveTasks(filePath string, tasks []Task) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	return gob.NewEncoder(file).Encode(tasks)
}
