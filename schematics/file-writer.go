package schematics

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
)

type FileApplyWriter struct {
	targetFolder string
}

func (fw *FileApplyWriter) TargetFolder() string {
	return fw.targetFolder
}

func (fw *FileApplyWriter) FileExists(fn string) bool {
	return fileutil.FileExists(fn)
}

func (fw *FileApplyWriter) WriteFile(fn string, p []byte) error {

	dir := filepath.Dir(fn)
	if !fileutil.FileExists(dir) {
		err := os.MkdirAll(dir, fs.ModePerm)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(fn, p, fs.ModePerm)
}

func (fw *FileApplyWriter) ListFiles(rexp *regexp.Regexp) (map[string]struct{}, error) {
	const semLogContext = "file-apply-writer::list-files"
	files, err := fileutil.FindFiles(fw.targetFolder, fileutil.WithFindOptionNavigateSubDirs(), fileutil.WithFindFileType(fileutil.FileTypeFile))
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	var m map[string]struct{}
	if len(files) > 0 {
		m = make(map[string]struct{})
	}

	for _, f := range files {
		// consider only the files that are matched by the regexp or all the files if the expression is nil
		if rexp == nil || (rexp != nil && rexp.Match([]byte(f))) {
			log.Trace().Str("fn", f).Msg(semLogContext)
			m[f] = struct{}{}
		}
	}

	return m, nil
}

func (fw *FileApplyWriter) RecoverRegionsOfFile(fromFile string, toContent []byte) ([]byte, error) {
	if !fileutil.FileExists(fromFile) {
		return toContent, nil
	}

	b, err := os.ReadFile(fromFile)
	if err != nil {
		return nil, err
	}

	return RecoverRegions(b, toContent)
}

func (fw *FileApplyWriter) ReadFile(fn string) ([]byte, error) {
	return os.ReadFile(fn)
}
