package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
)

// fakeReleases serves a GitHub-like releases page with one release, tag, whose
// archive for this platform contains bin.
func fakeReleases(t *testing.T, tag string, bin []byte, badChecksum bool) *httptest.Server {
	t.Helper()
	name := archiveName(runtime.GOOS, runtime.GOARCH)
	archive := makeArchive(t, name, bin)
	sum := sha256.Sum256(archive)
	if badChecksum {
		sum[0] ^= 0xff
	}
	sums := fmt.Sprintf("%x  %s\n%x  chekin_plan9_386.tar.gz\n", sum, name, sum)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/releases/tag/"+tag, http.StatusFound)
		case "/releases/download/" + tag + "/checksums.txt":
			w.Write([]byte(sums))
		case "/releases/download/" + tag + "/" + name:
			w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func makeArchive(t *testing.T, name string, bin []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if strings.HasSuffix(name, ".zip") {
		zw := zip.NewWriter(&buf)
		f, _ := zw.Create("chekin.exe")
		f.Write(bin)
		zw.Close()
		return buf.Bytes()
	}
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "README.md", Mode: 0o644, Size: 2, Typeflag: tar.TypeReg})
	tw.Write([]byte("hi"))
	tw.WriteHeader(&tar.Header{Name: "chekin", Mode: 0o755, Size: int64(len(bin)), Typeflag: tar.TypeReg})
	tw.Write(bin)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

// asBuild pretends this binary is version v of the given kind.
func asBuild(t *testing.T, v, kind string) {
	t.Helper()
	oldV, oldK := Version, buildKind
	Version, buildKind = v, kind
	t.Cleanup(func() { Version, buildKind = oldV, oldK })
}

func runUpdate(t *testing.T, srv *httptest.Server, exe string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	app := &App{Stdin: strings.NewReader(""), Stdout: &out, Stderr: &errb, releasesURL: srv.URL + "/releases", exePath: exe}
	code := app.Run(context.Background(), append([]string{"update"}, args...))
	return code, out.String(), errb.String()
}

func fakeExe(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "chekin")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestUpdateReplacesBinary(t *testing.T) {
	asBuild(t, "v0.1.0", buildRelease)
	srv := fakeReleases(t, "v0.2.0", []byte("new binary"), false)
	exe := fakeExe(t)

	code, out, stderr := runUpdate(t, srv, exe, "--yes")
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(out, "v0.1.0 -> v0.2.0") || !strings.Contains(stderr, "Updated") {
		t.Errorf("out=%q stderr=%q", out, stderr)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "new binary" {
		t.Errorf("binary = %q", got)
	}
	if fi, _ := os.Stat(exe); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v", fi.Mode())
	}
}

func TestUpdateRejectsBadChecksum(t *testing.T) {
	asBuild(t, "v0.1.0", buildRelease)
	srv := fakeReleases(t, "v0.2.0", []byte("new binary"), true)
	exe := fakeExe(t)

	code, _, stderr := runUpdate(t, srv, exe, "--yes")
	if code != 1 || !strings.Contains(stderr, "checksum") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Errorf("binary was replaced: %q", got)
	}
}

func TestUpdateCheckOnly(t *testing.T) {
	asBuild(t, "v0.1.0", buildRelease)
	srv := fakeReleases(t, "v0.2.0", []byte("new binary"), false)
	exe := fakeExe(t)

	code, out, _ := runUpdate(t, srv, exe, "--check")
	if code != 0 || !strings.Contains(out, "Update available: v0.1.0 -> v0.2.0") {
		t.Fatalf("code=%d out=%s", code, out)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Errorf("--check replaced the binary")
	}
}

func TestUpdateAlreadyLatest(t *testing.T) {
	asBuild(t, "v0.2.0", buildRelease)
	srv := fakeReleases(t, "v0.2.0", nil, false)
	code, out, _ := runUpdate(t, srv, fakeExe(t))
	if code != 0 || !strings.Contains(out, "v0.2.0 is up to date") {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestUpdateNeedsYesWithoutTerminal(t *testing.T) {
	asBuild(t, "v0.1.0", buildRelease)
	srv := fakeReleases(t, "v0.2.0", []byte("new binary"), false)
	code, _, stderr := runUpdate(t, srv, fakeExe(t))
	if code != 2 || !strings.Contains(stderr, "--yes") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func TestUpdateLeavesOtherInstallsAlone(t *testing.T) {
	for _, tc := range []struct{ kind, want string }{
		{buildGoInstall, "go install github.com/wilfred93/chekin-cli/cmd/chekin@latest"},
		{buildDev, "development build"},
	} {
		asBuild(t, "v0.1.0", tc.kind)
		srv := fakeReleases(t, "v0.2.0", []byte("new binary"), false)
		exe := fakeExe(t)
		code, _, stderr := runUpdate(t, srv, exe, "--yes")
		if code != 1 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: code=%d stderr=%s", tc.kind, code, stderr)
		}
		if got, _ := os.ReadFile(exe); string(got) != "old binary" {
			t.Errorf("%s: binary was replaced", tc.kind)
		}
	}
}

func TestUpdateNoReleases(t *testing.T) {
	asBuild(t, "v0.1.0", buildRelease)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/releases", http.StatusFound) // what GitHub does with no releases
	}))
	t.Cleanup(srv.Close)
	code, _, stderr := runUpdate(t, srv, fakeExe(t), "--check")
	if code != 1 || !strings.Contains(stderr, "no published release") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func TestDetectBuild(t *testing.T) {
	vcs := []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}}
	for _, tc := range []struct {
		name, ld  string
		bi        *debug.BuildInfo
		wantV, wk string
	}{
		{"release", "v0.3.0", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "v0.3.0", buildRelease},
		{"go install", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.3.0"}}, "v0.3.0", buildGoInstall},
		{"checkout", "dev", &debug.BuildInfo{Main: debug.Module{Version: "v0.3.1-0.20261104-abc+dirty"}, Settings: vcs}, "v0.3.1-0.20261104-abc+dirty", buildDev},
		{"no version", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, "dev", buildDev},
		{"no build info", "dev", nil, "dev", buildDev},
	} {
		v, k := detectBuild(tc.ld, tc.bi)
		if v != tc.wantV || k != tc.wk {
			t.Errorf("%s: got %q %q, want %q %q", tc.name, v, k, tc.wantV, tc.wk)
		}
	}
}

func TestChecksumFor(t *testing.T) {
	sums := []byte("aa  chekin_linux_amd64.tar.gz\nBB *chekin_windows_amd64.zip\n")
	if got := checksumFor(sums, "chekin_windows_amd64.zip"); got != "bb" {
		t.Errorf("got %q", got)
	}
	if got := checksumFor(sums, "chekin_darwin_arm64.tar.gz"); got != "" {
		t.Errorf("got %q", got)
	}
}
