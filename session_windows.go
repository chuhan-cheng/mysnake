package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

var (
	clock            roundClock
	history          []roundRecord
	dataDirectory    string
	recordPath       string
	saveError        bool
	lastSave         time.Time
	showHistory      bool
	historyPage      int
	muted            bool
	newBest          bool
	foodBonus        int
	foodAt, impactAt time.Time
)

func openHistory() syscall.Handle {
	root, err := os.UserConfigDir()
	if err != nil {
		saveError = true
		return 0
	}
	directory := filepath.Join(root, "MySnake")
	if err := os.MkdirAll(directory, 0700); err != nil {
		saveError = true
		return 0
	}
	// Exclusive access prevents a second window from recovering a live round.
	lockPath := wide(filepath.Join(directory, "session.lock"))
	handle, err := syscall.CreateFile(lockPath, syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		saveError = true
		return 0
	}
	dataDirectory = directory
	history, err = loadRecords(filepath.Join(directory, "rounds"))
	saveError = err != nil
	current.best = bestFruitScore(history)
	if data, err := os.ReadFile(filepath.Join(directory, "muted.json")); err == nil {
		muted = string(data) == "true\n"
	}
	return handle
}

func checkpoint(now time.Time) {
	if recordPath == "" {
		return
	}
	record := clock.record
	record.Duration = clock.elapsed(now)
	record.Score = current.score
	saveError = saveJSON(recordPath, record) != nil
	lastSave = now
}

func finishRound(now time.Time, result string) {
	if clock.record.Result != inProgress {
		return
	}
	clock.pause(now)
	clock.record.Result = result
	clock.record.Score = current.score
	checkpoint(now)
	history = append([]roundRecord{clock.record}, history...)
	if result != abandoned && current.score > current.best {
		current.best = current.score
		newBest = true
	}
}

func restart(now time.Time) {
	finishRound(now, abandoned)
	current.reset()
	clock = roundClock{}
	recordPath = ""
	newBest = false
	foodBonus = 0
	foodAt, impactAt = time.Time{}, time.Time{}
}

func pauseRound(now time.Time) {
	current.paused = true
	clock.pause(now)
	checkpoint(now)
}

func advance(now time.Time) {
	if !current.started || current.paused || current.ended {
		return
	}
	started := clock.record.Started.IsZero()
	if started {
		clock = roundClock{record: roundRecord{Started: now, Result: inProgress, Rules: fruitRules}, resumed: now}
		if dataDirectory != "" {
			recordPath = filepath.Join(dataDirectory, "rounds", fmt.Sprintf("%d.json", now.UnixNano()))
		}
	}
	score := current.score
	current.step()
	if current.score != score {
		foodAt = now
		foodBonus = current.score - score
		playPickup()
	}
	if current.ended {
		result := collision
		if current.won {
			result = completed
		}
		finishRound(now, result)
		if !current.won {
			impactAt = now
			if newBest {
				playCue("SystemAsterisk")
			} else {
				playCue("SystemExclamation")
			}
		}
		return
	}
	if started || score != current.score || now.Sub(lastSave) >= 5*time.Second {
		checkpoint(now)
	}
}

var winmm = syscall.NewLazyDLL("winmm.dll")
var playSound = winmm.NewProc("PlaySoundW")

func playCue(alias string) {
	if !muted {
		playSound.Call(uintptr(unsafe.Pointer(wide(alias))), 0, 0x10003)
	}
}

func toggleMute() {
	muted = !muted
	if muted {
		stopAudio()
		playSound.Call(0, 0, 0)
	} else {
		startAudio()
		playPickup()
	}
	if dataDirectory != "" && saveJSON(filepath.Join(dataDirectory, "muted.json"), muted) != nil {
		saveError = true
	}
}
