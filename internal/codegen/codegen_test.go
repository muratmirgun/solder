package codegen

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the want_* golden files")

// Each testdata/<case> is copied into a temporary module example.com/app
// that requires this repository. A case has want_err.txt when generation
// must fail; otherwise want_gen.go is the generated file, want_graph.mmd its
// `solder graph` output and want_out.txt the output of `go run .`.
func TestGolden(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	cases, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		src := filepath.Join("testdata", c.Name())
		t.Run(c.Name(), func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			copyCase(t, src, dir)
			gomod := "module example.com/app\n\ngo 1.26.0\n\nrequire github.com/muratmirgun/solder v0.0.0\n\nreplace github.com/muratmirgun/solder => " + root + "\n"
			write(t, filepath.Join(dir, "go.mod"), []byte(gomod))
			write(t, filepath.Join(dir, "go.sum"), sum)

			files, genErr := Generate(dir, "./...")
			wantErr := filepath.Join(src, "want_err.txt")
			if _, err := os.Stat(wantErr); err == nil || (*update && genErr != nil) {
				if genErr == nil {
					t.Fatalf("generation succeeded, want an error")
				}
				golden(t, wantErr, []byte(genErr.Error()+"\n"))
				return
			}
			if genErr != nil {
				t.Fatalf("generate: %v", genErr)
			}
			if len(files) != 1 {
				t.Fatalf("got %d files, want 1", len(files))
			}
			golden(t, filepath.Join(src, "want_gen.go"), files[0].Content)
			graph, err := Graph(dir, "./...")
			if err != nil {
				t.Fatalf("graph: %v", err)
			}
			golden(t, filepath.Join(src, "want_graph.mmd"), []byte(graph))
			write(t, files[0].Path, files[0].Content)
			cmd := exec.Command("go", "run", ".")
			cmd.Dir = dir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go run: %v\n%s", err, out)
			}
			golden(t, filepath.Join(src, "want_out.txt"), out)
		})
	}
}

func copyCase(t *testing.T, src, dst string) {
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasPrefix(d.Name(), "want_") {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(dst, filepath.Dir(rel)), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func golden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *update {
		write(t, path, got)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create it); got:\n%s", err, got)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s differs\n--- got\n%s\n--- want\n%s", path, got, want)
	}
}

func TestLowerFirst(t *testing.T) {
	for in, want := range map[string]string{"Config": "config", "DB": "db", "DBs": "dbs", "HTTPServer": "httpServer", "x": "x"} {
		if got := lowerFirst(in); got != want {
			t.Errorf("lowerFirst(%q) = %q, want %q", in, got, want)
		}
	}
}
