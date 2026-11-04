package cli

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	defaultReleasesURL = "https://github.com/wilfred93/chekin-cli/releases"
	goInstallCmd       = "go install github.com/wilfred93/chekin-cli/cmd/chekin@latest"
	maxDownload        = 100 << 20 // the archives are a few MB
)

func (a *App) update(ctx context.Context, args []string) error {
	fs := a.flags("update", "chekin update [--check] [--yes]")
	check := fs.Bool("check", false, "only report whether a newer release exists")
	yes := fs.Bool("yes", false, "update without asking")
	if err := parse(fs, args); err != nil {
		return err
	}

	latest, err := a.latestRelease(ctx)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	newer := buildKind == buildDev || semver.Compare(latest, Version) > 0
	if !newer {
		fmt.Fprintf(a.Stdout, "chekin %s is up to date.\n", Version)
		return nil
	}
	fmt.Fprintf(a.Stdout, "Update available: %s -> %s\n", Version, latest)
	if *check {
		return nil
	}

	switch buildKind {
	case buildGoInstall:
		return fmt.Errorf("this copy was installed with go install; update it with:\n  %s", goInstallCmd)
	case buildDev:
		return errors.New("this is a development build, so it won't be replaced; pull and rebuild, or use the install script")
	}

	exe, err := a.executable()
	if err != nil {
		return err
	}
	os.Remove(exe + ".old") // left behind by an earlier update on Windows
	if !*yes {
		if !a.interactive() {
			return usageErr("run `chekin update --yes` to update without a prompt")
		}
		fmt.Fprintf(a.Stderr, "Replace %s with %s? [y/N] ", exe, latest)
		answer, _ := readLine(bufio.NewReader(a.Stdin))
		if ans := strings.ToLower(answer); ans != "y" && ans != "yes" {
			fmt.Fprintln(a.Stderr, "Not updated.")
			return nil
		}
	}

	bin, err := a.downloadRelease(ctx, latest)
	if err != nil {
		return err
	}
	if err := replaceFile(exe, bin); err != nil {
		return err
	}
	fmt.Fprintf(a.Stderr, "Updated %s to %s.\n", exe, latest)
	return nil
}

func (a *App) releases() string {
	if a.releasesURL != "" {
		return a.releasesURL
	}
	return defaultReleasesURL
}

func (a *App) executable() (string, error) {
	if a.exePath != "" {
		return a.exePath, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("finding the chekin binary: %w", err)
	}
	return filepath.EvalSymlinks(exe)
}

func (a *App) httpGet(ctx context.Context, url string, follow bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "chekin-cli/"+Version)
	c := &http.Client{Timeout: 2 * time.Minute}
	if !follow {
		c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	}
	return c.Do(req)
}

// latestRelease returns the newest release's tag. GitHub redirects
// releases/latest to releases/tag/<tag>, which avoids the API's rate limit.
func (a *App) latestRelease(ctx context.Context) (string, error) {
	resp, err := a.httpGet(ctx, a.releases()+"/latest", false)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	loc := resp.Header.Get("Location")
	_, tag, ok := strings.Cut(loc, "/releases/tag/")
	if resp.StatusCode/100 != 3 || !ok {
		return "", fmt.Errorf("no published release found at %s", a.releases())
	}
	if !semver.IsValid(tag) {
		return "", fmt.Errorf("latest release has an unexpected tag %q", tag)
	}
	return tag, nil
}

// archiveName is the release asset for this platform, e.g. chekin_darwin_arm64.tar.gz.
func archiveName(goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return "chekin_" + goos + "_" + goarch + ext
}

// downloadRelease fetches this platform's archive for tag, checks it against
// the release's checksums.txt and returns the chekin binary inside it.
func (a *App) downloadRelease(ctx context.Context, tag string) ([]byte, error) {
	name := archiveName(runtime.GOOS, runtime.GOARCH)
	base := a.releases() + "/download/" + tag + "/"

	sums, err := a.download(ctx, base+"checksums.txt")
	if err != nil {
		return nil, err
	}
	want := checksumFor(sums, name)
	if want == "" {
		return nil, fmt.Errorf("release %s has no build for %s/%s", tag, runtime.GOOS, runtime.GOARCH)
	}
	archive, err := a.download(ctx, base+name)
	if err != nil {
		return nil, err
	}
	got := sha256.Sum256(archive)
	if hex.EncodeToString(got[:]) != want {
		return nil, fmt.Errorf("%s doesn't match its checksum; not updating", name)
	}
	return extractBinary(name, archive)
}

func (a *App) download(ctx context.Context, url string) ([]byte, error) {
	resp, err := a.httpGet(ctx, url, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownload+1))
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", url, err)
	}
	if len(data) > maxDownload {
		return nil, fmt.Errorf("downloading %s: file is too large", url)
	}
	return data, nil
}

// checksumFor finds name in sha256sum output ("<hex>  <name>" per line).
func checksumFor(sums []byte, name string) string {
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name {
			return strings.ToLower(f[0])
		}
	}
	return ""
}

func extractBinary(archiveName string, archive []byte) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".zip") {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if path.Base(f.Name) == "chekin.exe" {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(io.LimitReader(rc, maxDownload))
			}
		}
		return nil, fmt.Errorf("%s has no chekin.exe", archiveName)
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("%s has no chekin binary", archiveName)
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && path.Base(h.Name) == "chekin" {
			return io.ReadAll(io.LimitReader(tr, maxDownload))
		}
	}
}

// replaceFile swaps target for data with a rename, so a failed update leaves
// the old binary in place.
func replaceFile(target string, data []byte) error {
	hint := func(err error) error {
		if errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("can't replace %s: %w\nre-run with sudo, or reinstall into a directory you own", target, err)
		}
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), ".chekin-update-*")
	if err != nil {
		return hint(err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		// Windows can't overwrite a running .exe, but it can rename it.
		old := target + ".old"
		os.Remove(old)
		if err := os.Rename(target, old); err != nil {
			return hint(err)
		}
	}
	return hint(os.Rename(tmp.Name(), target))
}
