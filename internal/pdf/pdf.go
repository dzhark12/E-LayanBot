package pdf

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

//go:embed templates/*.tex templates/*.sty templates/assets/*
var templates embed.FS

var ErrNoTemplate = errors.New("belum ada template untuk jenis surat ini")

var fileByCode = map[string]string{
	"DM": "sk_domisili", "SU": "sk_usaha", "TM": "sk_tidak_mampu", "KM": "sk_kematian",
}

func Has(code string) bool { _, ok := fileByCode[code]; return ok }

var placeholder = regexp.MustCompile(`@@(\w+)@@`)

var latexEscape = strings.NewReplacer(
	`\`, `\textbackslash{}`, `&`, `\&`, `%`, `\%`, `$`, `\$`, `#`, `\#`, `_`, `\_`,
	`{`, `\{`, `}`, `\}`, `~`, `\textasciitilde{}`, `^`, `\textasciicircum{}`,
	"\n", " ", "\r", " ",
)

var sem = make(chan struct{}, 2) // maksimal 2 pembuatan PDF bersamaan

func Render(ctx context.Context, code string, data map[string]string) ([]byte, error) {
	name, ok := fileByCode[code]
	if !ok {
		return nil, ErrNoTemplate
	}
	src, err := templates.ReadFile("templates/" + name + ".tex")
	if err != nil {
		return nil, err
	}
	var missing []string
	filled := placeholder.ReplaceAllStringFunc(string(src), func(m string) string {
		key := placeholder.FindStringSubmatch(m)[1]
		v, ok := data[key]

		// Nomor surat boleh kosong.
		// Berikan jarak tetap pada PDF jika tidak diisi.
		if key == "NoUrut" && (!ok || strings.TrimSpace(v) == "") {
			return `\hspace*{1.5cm}`
		}

		// Semua placeholder lainnya tetap wajib diisi.
		if !ok || strings.TrimSpace(v) == "" {
			missing = append(missing, key)
			return ""
		}

		if key == "TtdJabatan" && ok && strings.TrimSpace(v) != "" {
			return v
		}

		return latexEscape.Replace(v)
	})
	if len(missing) > 0 {
		return nil, fmt.Errorf("data belum lengkap: %s", strings.Join(missing, ", "))
	}

	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	dir, err := os.MkdirTemp("", "surat-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	// salin desa.sty dan assets/ (semua selain .tex)
	err = fs.WalkDir(templates, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || strings.HasSuffix(p, ".tex") {
			return err
		}
		rel := strings.TrimPrefix(p, "templates")
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		target := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		b, err := templates.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o600)
	})
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, name+".tex"), []byte(filled), 0o600); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "pdflatex",
		"-interaction=nonstopmode", "-halt-on-error", "-no-shell-escape", name+".tex")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+dir, "TEXMFVAR="+dir, "TEXMFCONFIG="+dir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pembuatan PDF gagal")
	}
	return os.ReadFile(filepath.Join(dir, name+".pdf"))
}
