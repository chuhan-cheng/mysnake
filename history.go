package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	inProgress = "進行中"
	abandoned  = "中途退出"
	collision  = "撞擊結束"
	completed  = "填滿棋盤"
	fruitRules = 1
)

type roundRecord struct {
	Started  time.Time     `json:"started"`
	Duration time.Duration `json:"duration_ns"`
	Score    int           `json:"score"`
	Result   string        `json:"result"`
	Rules    int           `json:"rules,omitempty"`
}

func bestFruitScore(records []roundRecord) int {
	best := 0
	for _, record := range records {
		if record.Rules == fruitRules && (record.Result == collision || record.Result == completed) {
			best = max(best, record.Score)
		}
	}
	return best
}

type roundClock struct {
	record  roundRecord
	resumed time.Time
}

func (c *roundClock) pause(now time.Time) {
	c.record.Duration = c.elapsed(now)
	c.resumed = time.Time{}
}

func (c roundClock) elapsed(now time.Time) time.Duration {
	if c.resumed.IsZero() {
		return c.record.Duration
	}
	return c.record.Duration + now.Sub(c.resumed)
}

func loadRecords(directory string) ([]roundRecord, error) {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var history []roundRecord
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		var record roundRecord
		data, err := os.ReadFile(path)
		if err == nil {
			err = json.Unmarshal(data, &record)
		}
		if err != nil || record.Started.IsZero() {
			continue
		}
		if record.Result == inProgress {
			record.Result = abandoned
			// Keep the checkpoint readable even if recovery cannot rewrite it.
			_ = saveJSON(path, record)
		}
		history = append(history, record)
	}
	sort.Slice(history, func(i, j int) bool {
		return history[i].Started.After(history[j].Started)
	})
	return history, nil
}

func saveJSON(path string, value any) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".save-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	err = json.NewEncoder(file).Encode(value)
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
