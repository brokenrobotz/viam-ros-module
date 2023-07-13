package camera

import "fmt"

type ROSLidarConfig struct {
	NodeName   string `json:"node_name"`
	PrimaryUri string `json:"primary_uri"`
	Topic      string `json:"topic"`
}

func (cfg *ROSLidarConfig) Validate(path string) ([]string, error) {
	// NodeName will get default value if string is empty
	if cfg.PrimaryUri == "" {
		return nil, fmt.Errorf(`expected "PrimaryUri" attribute for sensor %q`, path)
	}

	if cfg.Topic == "" {
		return nil, fmt.Errorf(`expected "RosTopic" attribute for sensor %q`, path)
	}

	return nil, nil
}
