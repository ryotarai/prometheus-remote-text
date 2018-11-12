package main

import (
	"os"
	"time"
)

type TriggerFile struct {
	path        string
	lastModTime time.Time
}

func NewTriggerFile(path string) (*TriggerFile, error) {
	f := &TriggerFile{
		path: path,
	}
	t, err := f.modTime()
	if err != nil {
		return nil, err
	}
	f.lastModTime = t

	return f, nil
}

func (f *TriggerFile) CheckIfTouched() (bool, error) {
	t, err := f.modTime()
	if err != nil {
		return false, err
	}

	if t.After(f.lastModTime) {
		f.lastModTime = t
		return true, nil
	}
	return false, nil
}

func (f *TriggerFile) modTime() (time.Time, error) {
	zeroTime := time.Time{}

	i, err := os.Stat(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return zeroTime, nil
		}
		return zeroTime, err
	}
	return i.ModTime(), nil
}
