package state

import (
	"context"
	"os"
	"path"
)

type LocalManager struct {
	BaseDir   string
	StateFile string
}

func (m *LocalManager) stateFile() string {
	if m.StateFile == "" {
		return "state.json"
	}
	return m.StateFile
}

func (m *LocalManager) stateFilePath() string {
	return path.Join(m.BaseDir, m.stateFile())
}

func (m *LocalManager) Load(ctx context.Context) (*State, error) {
	data, err := os.ReadFile(m.stateFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &State{}, nil
		}
		return nil, err
	}

	return FromJSON(data)
}

func (m *LocalManager) Save(ctx context.Context, s *State) error {
	data, err := ToJSON(s)
	if err != nil {
		return err
	}
	return os.WriteFile(m.stateFilePath(), data, 0644)
}
