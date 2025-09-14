package testlogger

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

type msg struct {
	Message string `json:"message"`
}

func New() (zerolog.Logger, *BufWriter) {
	w := &BufWriter{
		b: strings.Builder{},
	}

	return zerolog.New(w), w
}

type BufWriter struct {
	b strings.Builder
}

func (w *BufWriter) Write(s []byte) (int, error) {
	outer := msg{}
	if err := json.Unmarshal(s, &outer); err != nil {
		return 0, fmt.Errorf("unmarshal 1: %w", err)
	}

	inner := make(map[string]interface{})
	if err := json.Unmarshal([]byte(outer.Message), &inner); err != nil {
		return 0, fmt.Errorf("unmarshal 2: %w", err)
	}

	out, err := json.Marshal(inner)
	if err != nil {
		return 0, fmt.Errorf("marshal: %w", err)
	}

	out = append(out, '\n')

	return w.b.Write(out)
}

func (w *BufWriter) Content() string {
	return w.b.String()
}
