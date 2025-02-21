package logic

import (
	"fmt"
	"os"
	"path/filepath"
	"rss_parrot/shared"
	"runtime"
	"runtime/pprof"
	"time"
)

const startDelaySec = 10
const profilerLoopSec = 60

type IProfiler interface {
}

type profiler struct {
	profileDir      string
	profileKeepDays int
}

func NewProfiler(cfg *shared.Config) IProfiler {
	prof := profiler{cfg.ProfileDir, cfg.ProfileKeepDays}
	go func() {
		time.Sleep(startDelaySec * time.Second)
		prof.profilerLoop()
	}()

	return &prof
}

func saveProfile(profileDir string) error {
	ts := time.Now().Format("2006-01-02!15-04-05")
	fname := fmt.Sprintf("%v.txt", ts)
	profPath := filepath.Join(profileDir, fname)
	f, err := os.Create(profPath)
	if err != nil {
		return err
	}
	defer f.Close()

	numGoroutine := runtime.NumGoroutine()
	if _, err = fmt.Fprintf(f, "Goroutine count: %d\n\n", numGoroutine); err != nil {
		return err
	}

	if err = pprof.Lookup("goroutine").WriteTo(f, 2); err != nil {
		return err
	}
	return nil
}

func purgeOld(profileDir string, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return filepath.Walk(profileDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.ModTime().Before(cutoff) {
			return os.Remove(path)
		}
		return nil
	})
}

func (prof *profiler) saveProfileAndPurgeOld() error {
	if err := saveProfile(prof.profileDir); err != nil {
		return err
	}
	if err := purgeOld(prof.profileDir, prof.profileKeepDays); err != nil {
		return err
	}
	return nil
}

func (prof *profiler) profilerLoop() {
	for {
		_ = prof.saveProfileAndPurgeOld()
		time.Sleep(profilerLoopSec * time.Second)
	}
}
