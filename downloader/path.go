package downloader

import (
	"bytes"
	"fmt"
	"iwaradl/api"
	"iwaradl/config"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/flytam/filenamify"
)

const defaultFilenameTemplate = "{{title}}-{{video_id}}"

type OutputPath struct {
	Dir      string
	FilePath string
}

func ResolveOutputPath(vi api.VideoInfo, quality string, downloadDirTemplate string, filenameTemplate string) (OutputPath, error) {
	dir, err := resolveDownloadPath(downloadDirTemplate, vi, quality)
	if err != nil {
		return OutputPath{}, err
	}
	stem := renderFilenameStem(vi, quality, filenameTemplate)
	filename := filepath.Join(dir, stem+".mp4")
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		return OutputPath{}, err
	}
	return OutputPath{Dir: dir, FilePath: absFilename}, nil
}

func resolveDownloadPath(pathTpl string, vi api.VideoInfo, quality string) (string, error) {
	if strings.TrimSpace(pathTpl) == "" {
		p := PrepareFolder(vi.User.Name)
		absPath, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return "", err
		}
		return absPath, nil
	}

	ctx := templateContext(vi, quality)
	t, err := templateWithVars("download_dir", ctx).Option("missingkey=default").Parse(pathTpl)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, ctx); err != nil {
		return "", err
	}

	raw := strings.TrimSpace(buf.String())
	if raw == "" {
		return "", fmt.Errorf("download_dir resolves to empty path")
	}
	resolved := filepath.Clean(raw)
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(config.Cfg.RootDir, resolved)
		resolved = filepath.Clean(resolved)
	}
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return "", err
	}
	return absPath, nil
}

func renderFilenameStem(vi api.VideoInfo, quality string, tpl string) string {
	ctx := templateContext(vi, quality)
	if strings.TrimSpace(tpl) == "" {
		tpl = config.Cfg.FilenameTemplate
	}
	if strings.TrimSpace(tpl) == "" {
		tpl = defaultFilenameTemplate
	}

	t, err := templateWithVars("filename", ctx).Option("missingkey=default").Parse(tpl)
	if err != nil {
		return fallbackFilenameStem(vi)
	}

	buf := bytes.NewBuffer(nil)
	if err := t.Execute(buf, ctx); err != nil {
		return fallbackFilenameStem(vi)
	}

	stem := strings.TrimSpace(buf.String())
	if stem == "" {
		return fallbackFilenameStem(vi)
	}
	cleaned, _ := filenamify.Filenamify(stem, filenamify.Options{Replacement: "_", MaxLength: 128})
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return fallbackFilenameStem(vi)
	}
	return cleaned
}

func templateWithVars(name string, ctx map[string]string) *template.Template {
	funcs := template.FuncMap{}
	for k, v := range ctx {
		val := v
		funcs[k] = func() string { return val }
	}
	return template.New(name).Funcs(funcs)
}

func templateContext(vi api.VideoInfo, quality string) map[string]string {
	return map[string]string{
		"now":             time.Now().Format("2006-01-02"),
		"publish_time":    vi.CreatedAt.Format("2006-01-02"),
		"title":           vi.Title,
		"video_id":        vi.Id,
		"author":          vi.User.Username,
		"author_nickname": vi.User.Name,
		"quality":         quality,
	}
}

func fallbackFilenameStem(vi api.VideoInfo) string {
	titleSafe, _ := filenamify.Filenamify(vi.Title, filenamify.Options{Replacement: "_", MaxLength: 64})
	if strings.TrimSpace(titleSafe) == "" {
		return vi.Id
	}
	return titleSafe + "-" + vi.Id
}
