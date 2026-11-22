package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRoundClockExcludesPause(t *testing.T) {
	start := time.Unix(100, 0).UTC()
	clock := roundClock{resumed: start}
	clock.pause(start.Add(30 * time.Second))
	if got := clock.elapsed(start.Add(40 * time.Second)); got != 30*time.Second {
		t.Fatalf("暫停時長為 %v，預期 30 秒", got)
	}
	clock.resumed = start.Add(40 * time.Second)
	if got := clock.elapsed(start.Add(60 * time.Second)); got != 50*time.Second {
		t.Fatalf("遊玩時長為 %v，預期 50 秒", got)
	}
}

func TestHistoryRecoversLatestCheckpoint(t *testing.T) {
	dir := t.TempDir()
	start := time.Unix(100, 0).UTC()
	old := roundRecord{Started: start, Score: 2, Result: collision}
	live := roundRecord{Started: start.Add(time.Minute), Result: inProgress}
	livePath := filepath.Join(dir, "new.json")
	if err := saveJSON(filepath.Join(dir, "old.json"), old); err != nil {
		t.Fatal(err)
	}
	if err := saveJSON(livePath, live); err != nil {
		t.Fatal(err)
	}
	live.Score, live.Duration = 5, 30*time.Second
	if err := saveJSON(livePath, live); err != nil {
		t.Fatal(err)
	}
	live.Result = abandoned
	for range 2 {
		records, err := loadRecords(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(records) != 2 || records[0] != live || records[1] != old {
			t.Fatalf("讀到的紀錄為 %+v，預期為 %+v、%+v", records, live, old)
		}
	}
}
