package export

import "encoding/json"

type SpecExportData[Spec any] struct {
	RelativePath string
	Data         *Spec
}

func (s *SpecExportData[Spec]) ToMap() (map[string]any, error) {
	bytes, err := json.Marshal(s.Data)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(bytes, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
