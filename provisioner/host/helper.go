package host

import (
	"context"
	"encoding/json"
	"fmt"
)

type ConfigResponse struct {
	Defer         bool              `json:"defer"`
	Msg           string            `json:"msg"`
	Certificate   string            `json:"certificate"`
	CA            string            `json:"ca"`
	Configuration map[string]string `json:"configuration"`
}

func (h *Host) getConfig(ctx context.Context) (*ConfigResponse, error) {
	r := &ConfigResponse{}

	input, err := json.Marshal(h)
	if err != nil {
		return nil, fmt.Errorf("could not JSON encode host: %s", err)
	}

	err = runDecodedHelper(ctx, h.cfg.Helper, []string{}, string(input), r)
	if err != nil {
		return nil, fmt.Errorf("could not invoke configure helper: %s", err)
	}

	return r, nil
}
