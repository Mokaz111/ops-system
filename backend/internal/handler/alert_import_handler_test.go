package handler

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"
)

const sampleRuleYaml = `groups:
  - name: node
    rules:
      - alert: HighCPU
        expr: node_cpu_usage > 0.9
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: CPU high
      - record: job:cpu:rate5m
        expr: rate(node_cpu_seconds_total[5m])
`

func TestExtractRuleFilesYaml(t *testing.T) {
	files, err := extractRuleFiles("rules.yaml", []byte(sampleRuleYaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0].Name != "rules.yaml" {
		t.Fatalf("expected single yaml file, got %+v", files)
	}
}

func TestExtractRuleFilesUnsupported(t *testing.T) {
	if _, err := extractRuleFiles("rules.json", []byte("{}")); err == nil {
		t.Fatal("expected error for unsupported extension")
	}
}

func TestExtractRuleFilesZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"a.yaml", "sub/b.yml", "README.md", ".hidden.yaml"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(sampleRuleYaml)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	files, err := extractRuleFiles("bundle.zip", buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 yaml entries (md/hidden skipped), got %d", len(files))
	}
}

func TestExtractRuleFilesTarGz(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	content := []byte(sampleRuleYaml)
	for _, name := range []string{"rules/a.yaml", "rules/notes.txt"} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	files, err := extractRuleFiles("bundle.tar.gz", buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0].Name != "rules/a.yaml" {
		t.Fatalf("expected only rules/a.yaml, got %+v", files)
	}
}
