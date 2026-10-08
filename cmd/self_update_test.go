package cmd

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseArchiveName(t *testing.T) {
	if got, want := releaseArchiveName("1.6.0", "linux", "amd64"), "github-pokemon_1.6.0_linux_amd64.tar.gz"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := releaseArchiveName("1.6.0", "windows", "arm64"), "github-pokemon_1.6.0_windows_arm64.zip"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("archive-bytes")
	sum := sha256.Sum256(data)
	good := hex.EncodeToString(sum[:]) + "  a.tar.gz\n" + "deadbeef  b.tar.gz\n"

	if err := verifyChecksum([]byte(good), "a.tar.gz", data); err != nil {
		t.Errorf("expected match, got %v", err)
	}
	if err := verifyChecksum([]byte(good), "b.tar.gz", data); err == nil {
		t.Error("expected mismatch error")
	}
	if err := verifyChecksum([]byte(good), "c.tar.gz", data); err == nil {
		t.Error("expected missing-entry error")
	}
}

func TestExtractBinaryTarGz(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{"README.md": "readme", "github-pokemon": "BINARY"} {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg})
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()

	got, err := extractBinary(buf.Bytes(), "x.tar.gz", "github-pokemon")
	if err != nil || string(got) != "BINARY" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := extractBinary(buf.Bytes(), "x.tar.gz", "missing"); err == nil {
		t.Error("expected not-found error")
	}
}

func TestExtractBinaryZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("github-pokemon.exe")
	_, _ = w.Write([]byte("EXE"))
	_ = zw.Close()

	got, err := extractBinary(buf.Bytes(), "x.zip", "github-pokemon.exe")
	if err != nil || string(got) != "EXE" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestReplaceExecutable(t *testing.T) {
	for _, goos := range []string{"linux", "windows"} {
		dir := t.TempDir()
		exe := filepath.Join(dir, "github-pokemon")
		if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := replaceExecutable(exe, []byte("new"), goos); err != nil {
			t.Fatalf("%s: %v", goos, err)
		}
		got, _ := os.ReadFile(exe)
		if string(got) != "new" {
			t.Errorf("%s: got %q", goos, got)
		}
		if info, _ := os.Stat(exe); info.Mode().Perm()&0o111 == 0 {
			t.Errorf("%s: new binary is not executable", goos)
		}

		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.Name() != "github-pokemon" && e.Name() != "github-pokemon.old" {
				t.Errorf("%s: leftover file %s", goos, e.Name())
			}
		}
	}
}
