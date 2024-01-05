package schematics

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	godiffpatch "github.com/sourcegraph/go-diff-patch"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	defaultConflictMode string
	produceDiff         bool
	onConflictPolicies  []ConflictPolicy
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

func Apply(targetFolder string, files []OpNode, opts ...ApplyOption) error {

	const semLogContext = "schematics::apply"

	cfg := ApplyOptions{}
	for _, o := range opts {
		o(&cfg)
	}

	var mergedFiles []OpNode
	for _, f := range files {
		targetPath := filepath.Join(targetFolder, f.path)
		cm, err := computeConflictMode(&cfg, targetPath)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}

		switch cm {
		case ConflictModeOverwrite:
			mergedFiles = append(mergedFiles, OpNode{path: targetPath, content: f.content})
		case ConflictModeKeep:
			// The file is not created. The previous is kept.
		case ConflictModeBackup:
			pf, err := createPatchFile(targetPath, f.content)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}

			// if files are not different... nothing happens.
			if !pf.IsZero() {
				// Since they are different it does make sense to produce the new file.
				mergedFiles = append(mergedFiles, OpNode{path: targetPath, content: f.content})

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
			pf, err := createPatchFile(targetPath, f.content)
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

				newf, err := createNewFile(targetPath, f.content)
				if err != nil {
					log.Error().Err(err).Msg(semLogContext)
					return err
				}
				mergedFiles = append(mergedFiles, newf)
			}
		}
	}

	for _, mf := range mergedFiles {
		log.Info().Str("file-name", mf.path).Msg(semLogContext)
		dir := filepath.Dir(mf.path)
		if !util.FileExists(dir) {
			err := os.MkdirAll(dir, fs.ModePerm)
			if err != nil {
				log.Error().Err(err).Msg(semLogContext)
				return err
			}
		}

		err := os.WriteFile(mf.path, mf.content, fs.ModePerm)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return err
		}
	}

	return nil
}

func computeConflictMode(cfg *ApplyOptions, targetPath string) (string, error) {
	const semLogContext = "schematics::compute-conflict-mode"

	if util.FileExists(targetPath) {
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

	if util.FileExists(targetPath) {
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
			return OpNode{path: patchFile, content: []byte(patch)}, nil
		}
	}

	return OpNode{}, nil
}

func createBackupFile(targetPath string) (OpNode, error) {
	const semLogContext = "schematics::create-bak-file"

	if util.FileExists(targetPath) {
		current, err := os.ReadFile(targetPath)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return OpNode{}, err
		}

		bakFile := filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath)+".bak")
		log.Info().Str("bak-file", bakFile).Msg(semLogContext)
		return OpNode{path: bakFile, content: []byte(current)}, nil
	}

	return OpNode{}, nil
}

func createNewFile(targetPath string, content []byte) (OpNode, error) {
	const semLogContext = "schematics::create-new-file"
	newFile := filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath)+".new")
	log.Info().Str("new-file", newFile).Msg(semLogContext)
	return OpNode{path: newFile, content: []byte(content)}, nil
}
