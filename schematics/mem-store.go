package schematics

import (
	"errors"
	"regexp"

	"github.com/rs/zerolog/log"
)

type ApplyMemoryStore struct {
	targetFolder string
	m            map[string][]byte
}

func NewApplyMemoryStore(targetFolder string) *ApplyMemoryStore {
	return &ApplyMemoryStore{targetFolder: targetFolder}
}

func (fw *ApplyMemoryStore) Files() map[string][]byte {
	return fw.m
}

func (fw *ApplyMemoryStore) TargetFolder() string {
	return fw.targetFolder
}

func (fw *ApplyMemoryStore) FileExists(fn string) bool {
	if len(fw.m) == 0 {
		return false
	}

	_, ok := fw.m[fn]
	return ok
}

func (fw *ApplyMemoryStore) WriteFile(fn string, p []byte) error {
	if fw.m == nil {
		fw.m = make(map[string][]byte)
	}

	fw.m[fn] = p
	return nil
}

func (fw *ApplyMemoryStore) ListFilenames(rexp *regexp.Regexp) (map[string]struct{}, error) {
	const semLogContext = "apply-memory-store::list-file-names"

	if len(fw.m) == 0 {
		return nil, nil
	}

	files := make(map[string]struct{})
	for k, _ := range fw.m {
		if rexp == nil || rexp.Match([]byte(k)) {
			log.Trace().Str("fn", k).Msg(semLogContext)
			files[k] = struct{}{}
		}
	}

	return files, nil
}

func (fw *ApplyMemoryStore) RecoverRegionsOfFile(fromFile string, toContent []byte) ([]byte, error) {
	if !fw.FileExists(fromFile) {
		return toContent, nil
	}

	b, _ := fw.m[fromFile]
	return RecoverRegions(b, toContent)
}

func (fw *ApplyMemoryStore) ReadFile(fn string) ([]byte, error) {
	if !fw.FileExists(fn) {
		return nil, errors.New("file not found in memory writer: " + fn)
	}

	b, _ := fw.m[fn]

	return b, nil
}
