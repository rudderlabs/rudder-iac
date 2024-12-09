package state

import "encoding/json"

func ToJSON(state *State) (json.RawMessage, error) {
	return json.Marshal(state)
}

func FromJSON(data json.RawMessage) (*State, error) {
	state := &State{}
	err := json.Unmarshal(data, state)
	if err != nil {
		return nil, err
	}
	return state, nil
}
