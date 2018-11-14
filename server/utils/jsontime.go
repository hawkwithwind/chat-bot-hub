package utils

import (
	"strings"
	"time"
)

type JSONTime struct {
	time.Time
}

func (t JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Time.Format(time.RFC3339) + `"`), nil
}

func (t *JSONTime) UnmarshalJSON(buf []byte) error {
	tt, err := time.Parse(time.RFC3339, strings.Trim(string(buf), `"`))
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}
