package downloader

import (
	"bytes"
	"fmt"
	"iwaradl/api"
	"iwaradl/config"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/flytam/filenamify"
)

const defaultFilenameTemplate = "{{title}}-{{video_id}}"

var externalVarPattern = regexp.MustCompile(`%#([A-Za-z]+)(?::([^#%]*))?#%`)

type templateContext struct {
	Now            time.Time
	PublishTime    time.Time
	Title          string
	VideoID        string
	Author         string
	AuthorNickname string
	Quality        string
}

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

// ConvertExternalTemplate converts third-party (IwaraDownloadTool) placeholders to Go template syntax.
// Supported placeholders are %#NowTime#%, %#UploadTime#%, %#TITLE#%, %#ID#%, %#AUTHOR#%, %#ALIAS#%, %#QUALITY#%.
func ConvertExternalTemplate(tpl string) string {
	if strings.TrimSpace(tpl) == "" {
		return tpl
	}
	return externalVarPattern.ReplaceAllStringFunc(tpl, func(match string) string {
		parts := externalVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		name := strings.ToUpper(parts[1])
		format := ""
		if len(parts) >= 3 {
			format = strings.TrimSpace(parts[2])
		}

		switch name {
		case "NOWTIME":
			layout := "2006-01-02"
			if format != "" {
				layout = externalTimeLayout(format)
			}
			return fmt.Sprintf("{{now %q}}", layout)
		case "UPLOADTIME":
			layout := "2006-01-02"
			if format != "" {
				layout = externalTimeLayout(format)
			}
			return fmt.Sprintf("{{publish_time %q}}", layout)
		case "TITLE":
			return "{{title}}"
		case "ID":
			return "{{video_id}}"
		case "AUTHOR":
			return "{{author}}"
		case "ALIAS":
			return "{{author_nickname}}"
		case "QUALITY":
			return "{{quality}}"
		default:
			return match
		}
	})
}

func externalTimeLayout(format string) string {
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"MM", "01",
		"DD", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
	)
	return replacer.Replace(format)
}

func resolveDownloadPath(pathTpl string, vi api.VideoInfo, quality string) (string, error) {
	pathTpl = ConvertExternalTemplate(pathTpl)
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

	ctx := makeTemplateContext(vi, quality)
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
	tpl = ConvertExternalTemplate(tpl)
	ctx := makeTemplateContext(vi, quality)
	if strings.TrimSpace(tpl) == "" {
		tpl = config.Cfg.FilenameTemplate
	}
	tpl = ConvertExternalTemplate(tpl)
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

func templateWithVars(name string, ctx templateContext) *template.Template {
	defaultLayout := "2006-01-02"
	formatWithDefault := func(t time.Time, layout []string) string {
		if len(layout) > 0 && strings.TrimSpace(layout[0]) != "" {
			return t.Format(layout[0])
		}
		return t.Format(defaultLayout)
	}

	funcs := template.FuncMap{
		"now": func(layout ...string) string {
			return formatWithDefault(ctx.Now, layout)
		},
		"publish_time": func(layout ...string) string {
			return formatWithDefault(ctx.PublishTime, layout)
		},
		"title":           func() string { return ctx.Title },
		"video_id":        func() string { return ctx.VideoID },
		"author":          func() string { return ctx.Author },
		"author_nickname": func() string { return ctx.AuthorNickname },
		"quality":         func() string { return ctx.Quality },
	}

	return template.New(name).Funcs(funcs)
}

func makeTemplateContext(vi api.VideoInfo, quality string) templateContext {
	return templateContext{
		Now:            time.Now(),
		PublishTime:    vi.CreatedAt,
		Title:          vi.Title,
		VideoID:        vi.Id,
		Author:         vi.User.Username,
		AuthorNickname: vi.User.Name,
		Quality:        quality,
	}
}

func fallbackFilenameStem(vi api.VideoInfo) string {
	titleSafe, _ := filenamify.Filenamify(vi.Title, filenamify.Options{Replacement: "_", MaxLength: 64})
	if strings.TrimSpace(titleSafe) == "" {
		return vi.Id
	}
	return titleSafe + "-" + vi.Id
}
