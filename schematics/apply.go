package schematics

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
	godiffpatch "github.com/sourcegraph/go-diff-patch"
)

const (
	ConflictModeOverwrite = "overwrite"
	ConflictModeKeep      = "keep"
	ConflictModeBackup    = "backup"
	ConflictModeNew       = "new"
)

type ConflictPolicy struct {
	mode        string
	includeList []*regexp.Regexp
}

type ApplyOptions struct {
	defaultConflictMode     string
	produceDiff             bool
	onConflictPolicies      []ConflictPolicy
	deleteOtherFiles        bool
	deleteOtherFilesPattern *regexp.Regexp
}

type ApplyOption func(*ApplyOptions)

func WithApplyProduceDiff() ApplyOption {
	return func(aopts *ApplyOptions) {
		aopts.produceDiff = true
	}
}

func WithApplyDefaultConflictMode(m string) ApplyOption {
	return func(aopts *ApplyOptions) {
		aopts.defaultConflictMode = m
	}
}

func WithApplyConflictPolicy(m string, include []string) ApplyOption {
	return func(aopts *ApplyOptions) {
		if len(include) > 0 {
			cp := ConflictPolicy{mode: m}
			for _, s := range include {
				cp.includeList = append(cp.includeList, regexp.MustCompile(s))
			}
		}
	}
}

func WithDeleteOtherFiles(pattern string) ApplyOption {
	return func(aopts *ApplyOptions) {
		if pattern != "" {
			aopts.deleteOtherFilesPattern = regexp.MustCompile(pattern)
		}
		aopts.deleteOtherFiles = true
	}
}

func Apply(targetFolder string, files []OpNode, opts ...ApplyOption) error {

	const semLogContext = "schematics::apply"

	cfg := ApplyOptions{}
	for _, o := range opts {
		o(&cfg)
	}

	var otherFiles map[string]struct{}
	var err error
	if cfg.deleteOtherFiles {
		otherFiles, err = findFilesInTargetFolder(targetFolder, cfg.deleteOtherFilesPattern)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}

		log.Info().Int("num-files", len(otherFiles)).Msg(semLogContext + " files under target folder")
	}

	var mergedFiles []OpNode
	for _, f := range files {
		if len(otherFiles) > 0 {
			fullPath := filepath.Join(targetFolder, f.Path)
			if _, ok := otherFiles[fullPath]; ok {
				log.Trace().Str("file-name", fullPath).Msg(semLogContext + " clear from map")
				delete(otherFiles, fullPath)
			}
		}

		targetPath := filepath.Join(targetFolder, f.Path)
		if fileutil.FileExists(targetPath) {
			log.Info().Str("path", targetPath).Msg(semLogContext + " - recovering regions")
			b, err := RecoverRegionsOfFile(targetPath, f.Content)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}
			f.Content = b
		}

		cm, err := computeConflictMode(&cfg, targetPath)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}

		switch cm {
		case ConflictModeOverwrite:
			mergedFiles = append(mergedFiles, OpNode{Path: targetPath, Content: f.Content})
		case ConflictModeKeep:
			// The file is not created. The previous is kept.
		case ConflictModeBackup:
			pf, err := createPatchFile(targetPath, f.Content)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}

			// if files are not different... nothing happens.
			if !pf.IsZero() {
				// Since they are different it does make sense to produce the new file.
				mergedFiles = append(mergedFiles, OpNode{Path: targetPath, Content: f.Content})

				// files are different. check if the patch file has to be produced.
				if cfg.produceDiff {
					mergedFiles = append(mergedFiles, pf)
				} else {
					log.Info().Msg(semLogContext + " actual patch creation not enabled")
				}

				bck, err := createBackupFile(targetPath)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return err
				}
				mergedFiles = append(mergedFiles, bck)
			}
		case ConflictModeNew:
			pf, err := createPatchFile(targetPath, f.Content)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}

			// if files are not different... nothing happens.
			if !pf.IsZero() {
				// files are different. check if the patch file has to be produced.
				if cfg.produceDiff {
					mergedFiles = append(mergedFiles, pf)
				} else {
					log.Info().Msg(semLogContext + " actual patch creation not enabled")
				}

				newf, err := createNewFile(targetPath, f.Content)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return err
				}
				mergedFiles = append(mergedFiles, newf)
			}
		}
	}

	for _, mf := range mergedFiles {
		log.Info().Str("file-name", mf.Path).Msg(semLogContext)
		dir := filepath.Dir(mf.Path)
		if !fileutil.FileExists(dir) {
			err := os.MkdirAll(dir, fs.ModePerm)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}
		}

		err := os.WriteFile(mf.Path, mf.Content, fs.ModePerm)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}
	}

	if len(otherFiles) > 0 {
		log.Info().Int("num-other-files", len(otherFiles)).Msg(semLogContext + " files not in current generation")
		for n, _ := range otherFiles {
			log.Info().Str("file-name", n).Msg(semLogContext + " file not in current generation")
			/*
				err = os.Rename(n, n+".del")
				if err != nil {
					log.Error().Err(err).Str("file-name", n).Msg(semLogContext)
				}
			*/
		}
	}

	return nil
}

func findFilesInTargetFolder(targetFolder string, rexp *regexp.Regexp) (map[string]struct{}, error) {
	const semLogContext = "schematics::find-targets"
	files, err := fileutil.FindFiles(targetFolder, fileutil.WithFindOptionNavigateSubDirs(), fileutil.WithFindFileType(fileutil.FileTypeFile))
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

func computeConflictMode(cfg *ApplyOptions, targetPath string) (string, error) {
	const semLogContext = "schematics::compute-conflict-mode"

	if fileutil.FileExists(targetPath) {
		baseName := filepath.Base(targetPath)
		for _, p := range cfg.onConflictPolicies {
			for _, r := range p.includeList {
				if r.Match([]byte(baseName)) {
					return p.mode, nil
				}
			}
		}

		return cfg.defaultConflictMode, nil
	}

	return ConflictModeOverwrite, nil
}

func createPatchFile(targetPath string, content []byte) (OpNode, error) {
	const semLogContext = "schematics::create-patch-file"

	if fileutil.FileExists(targetPath) {
		current, err := os.ReadFile(targetPath)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return OpNode{}, err
		}

		patch := godiffpatch.GeneratePatch(targetPath, string(current), string(content))
		if len(patch) > 0 {
			patchFile := filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath)+".patch")
			log.Info().Str("patch-file", patchFile).Msg(semLogContext)
			// _ = os.WriteFile(patchFile, []byte(patch), fs.ModePerm)
			return OpNode{Path: patchFile, Content: []byte(patch)}, nil
		}
	}

	return OpNode{}, nil
}

func createBackupFile(targetPath string) (OpNode, error) {
	const semLogContext = "schematics::create-bak-file"

	if fileutil.FileExists(targetPath) {
		current, err := os.ReadFile(targetPath)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return OpNode{}, err
		}

		bakFile := filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath)+".bak")
		log.Info().Str("bak-file", bakFile).Msg(semLogContext)
		return OpNode{Path: bakFile, Content: []byte(current)}, nil
	}

	return OpNode{}, nil
}

func createNewFile(targetPath string, content []byte) (OpNode, error) {
	const semLogContext = "schematics::create-new-file"
	newFile := filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath)+".new")
	log.Info().Str("new-file", newFile).Msg(semLogContext)
	return OpNode{Path: newFile, Content: []byte(content)}, nil
}
