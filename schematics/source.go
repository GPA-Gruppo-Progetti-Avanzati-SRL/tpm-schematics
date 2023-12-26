package schematics

import (
	"embed"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/templateutil"
	"github.com/rs/zerolog/log"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type SourceTemplateOptions struct {
	funcMap    template.FuncMap
	formatCode bool
	model      interface{}
	metadata   map[string]interface{}

	foldersIncludeList []string
	foldersIgnoreList  []string
	filesIncludeList   []string
	filesIgnoreList    []string
}

type SourceTemplateOption func(*SourceTemplateOptions)

func SourceWithFormatCode() SourceTemplateOption {
	return func(aopts *SourceTemplateOptions) {
		aopts.formatCode = true
	}
}

func SourceWithModel(m interface{}) SourceTemplateOption {
	return func(aopts *SourceTemplateOptions) {
		aopts.model = m
	}
}

func SourceWithMetadata(m map[string]interface{}) SourceTemplateOption {
	return func(aopts *SourceTemplateOptions) {
		aopts.metadata = m
	}
}

func SourceWithFuncMap(f template.FuncMap) SourceTemplateOption {
	return func(aopts *SourceTemplateOptions) {
		aopts.funcMap = f
	}
}

func WithSourceFindOptionFoldersIncludeList(p []string) SourceTemplateOption {
	return func(cfg *SourceTemplateOptions) {
		cfg.foldersIncludeList = p
	}
}

func WithSourceFindOptionFoldersIgnoreList(p []string) SourceTemplateOption {
	return func(cfg *SourceTemplateOptions) {
		cfg.foldersIgnoreList = p
	}
}

func WithSourceFindOptionFilesIncludeList(p []string) SourceTemplateOption {
	return func(cfg *SourceTemplateOptions) {
		cfg.filesIncludeList = p
	}
}

func WithSourceFindOptionFilesIgnoreList(p []string) SourceTemplateOption {
	return func(cfg *SourceTemplateOptions) {
		cfg.filesIgnoreList = p
	}
}

type SourceTemplate struct {
	path string
	// content   []byte
	templates []templateutil.Info
}

type OpNode struct {
	path    string
	content []byte
}

func (s *OpNode) IsZero() bool {
	return s.path == "" && s.content == nil
}

func (s *SourceTemplate) processTemplates(genCtx *SourceContext, funcMap template.FuncMap, formatCode bool) (OpNode, error) {
	const semLogContext = "schematics::process-template"

	var err error
	var out OpNode

	out.path, err = ResolveSchematicsName(s.path, genCtx.Metadata)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return out, err
	}

	var parsedTemplate *template.Template
	if parsedTemplate, err = templateutil.Parse(s.templates, funcMap); err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return out, err
	} else {
		if out.content, err = templateutil.Process(parsedTemplate, genCtx, formatCode); err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return out, err
		}
	}

	return out, nil
}

type SourceContext struct {
	Name       string
	Metadata   map[string]interface{}
	ProducedAt time.Time
	Model      interface{}
}

func GetSource(templates embed.FS, embedRootFolder string, opts ...SourceTemplateOption) ([]OpNode, error) {

	const semLogContext = "schematics::source"
	cfg := SourceTemplateOptions{}
	for _, o := range opts {
		o(&cfg)
	}

	nodes, err := readSourceTemplates(&cfg, templates, embedRootFolder)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	ctx := SourceContext{Name: cfg.metadata["name"].(string), ProducedAt: time.Now(), Model: cfg.model, Metadata: cfg.metadata}

	source, err := processSourceTemplates(&ctx, cfg.funcMap, nodes, cfg.formatCode)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return source, nil
}

func processSourceTemplates(ctx *SourceContext, funcMap template.FuncMap, nodes []SourceTemplate, formatCode bool) ([]OpNode, error) {
	const semLogContext = "schematics::process-source-templates"

	var opNodes []OpNode
	for _, n := range nodes {
		o, err := n.processTemplates(ctx, funcMap, formatCode)
		if err != nil {
			log.Error().Err(err).Msg(semLogContext)
			return nil, err
		}

		opNodes = append(opNodes, o)
	}

	return opNodes, nil
}

func readSourceTemplates(cfg *SourceTemplateOptions, templates embed.FS, rootFolder string) ([]SourceTemplate, error) {

	entries, err := util.FindEmbeddedFiles(
		templates, rootFolder,
		util.WithFindOptionNavigateSubDirs(), util.WithFindOptionExcludeRootFolderInNames(), util.WithFindOptionPreloadContent(),
		util.WithFindOptionFilesIncludeList(cfg.filesIncludeList), util.WithFindOptionFilesIgnoreList(cfg.filesIgnoreList),
		util.WithFindOptionFoldersIncludeList(cfg.foldersIncludeList), util.WithFindOptionFoldersIgnoreList(cfg.foldersIgnoreList))

	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, err
	}

	var treeNodes []SourceTemplate
	treeNodeMap := make(map[string]int)
	for _, e := range entries {
		if e.Info.IsDir() {
			continue
		}

		fn := e.Info.Name()
		var baseFn string
		if strings.HasSuffix(fn, ".template") {
			fn = strings.TrimSuffix(fn, ".template")
			baseFn = fn
		} else {
			if strings.HasSuffix(fn, ".child-template") {
				fn = strings.TrimSuffix(fn, ".child-template")
				ext := filepath.Ext(fn)
				if ext != "" {
					baseFn = strings.TrimSuffix(fn, ext)
				}
			} else {
				continue
			}
		}

		fulln := baseFn
		if e.Path != "" {
			fulln = filepath.Join(e.Path, baseFn)
		}

		if ndx, ok := treeNodeMap[baseFn]; ok {
			treeNodes[ndx].path = fulln
			treeNodes[ndx].templates = append(treeNodes[ndx].templates, templateutil.Info{Name: fn, Content: string(e.Content)})
		} else {
			treeNodes = append(treeNodes, SourceTemplate{
				path: fulln,
				templates: []templateutil.Info{
					{
						Name:    fn,
						Content: string(e.Content),
					},
				},
			})
			treeNodeMap[baseFn] = len(treeNodes) - 1
		}
	}

	return treeNodes, nil
}
