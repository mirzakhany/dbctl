package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const semVerRegex string = `v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?`

var (
	versionRegex = regexp.MustCompile("^" + semVerRegex + "$")
)

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Updater struct {
	releaseURL   string
	execFileName string
}

func New(owner, repo, execFileName string) *Updater {
	return &Updater{
		releaseURL:   fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo),
		execFileName: execFileName,
	}
}

func (u *Updater) Update(ctx context.Context) error {
	targetPath, err := os.Executable()
	if err != nil {
		return err
	}

	release, err := u.getLastRelease(ctx)
	if err != nil {
		return err
	}

	if len(release.Assets) != 1 {
		return errors.New("no release found for your platform")
	}

	asset := release.Assets[0]
	updateDir := filepath.Dir(targetPath)
	filename := filepath.Base(targetPath)

	// Download the latest update to tmp dir
	assetTarFile := path.Join("/tmp", asset.Name)
	if err := downloadAsset(ctx, asset.BrowserDownloadURL, path.Join("/tmp", asset.Name)); err != nil {
		return err
	}

	extIndex := strings.Index(asset.Name, ".")
	if extIndex != -1 {
		return errors.New("update failed to find path directory name")
	}

	downloadDest := path.Join("/tmp", asset.Name[:extIndex])
	if err := extractTarGz(assetTarFile, downloadDest); err != nil {
		return err
	}

	downloadedFile, err := os.Open(path.Join(downloadDest, u.execFileName))
	if err != nil {
		return err
	}
	defer downloadedFile.Close()

	oldFileInfo, err := os.Stat(targetPath)
	if err != nil {
		return err
	}

	// Copy the contents of new binary to a new executable file
	newBinary := filepath.Join(updateDir, fmt.Sprintf(".%s.new", filename))
	fp, err := os.OpenFile(newBinary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, oldFileInfo.Mode())
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = io.Copy(fp, downloadedFile)
	if err != nil {
		return err
	}
	fp.Close()
	downloadedFile.Close()

	// swap the binaries
	oldBinary := filepath.Join(updateDir, fmt.Sprintf(".%s.old", filename))
	_ = os.Remove(oldBinary)

	// move the existing executable to a new file in the same directory
	err = os.Rename(targetPath, oldBinary)
	if err != nil {
		return err
	}

	// move the new executable in to become the new program
	err = os.Rename(newBinary, targetPath)
	if err != nil {
		// move unsuccessful
		// Try to rollback by restoring the old binary to its original path.
		if rerr := os.Rename(oldBinary, targetPath); rerr != nil {
			return fmt.Errorf("rollback failed, %w", err)
		}
		return err
	}

	// Clean up download dir
	_ = os.RemoveAll(downloadDest)
	_ = os.RemoveAll(assetTarFile)
	return nil
}

func (u *Updater) LatestVersion(ctx context.Context) (*Version, error) {
	release, err := u.getLastRelease(ctx)
	if err != nil {
		return nil, err
	}

	if len(release.Assets) == 0 {
		return nil, errors.New("no release found for your platform")
	}

	return ParseVersion(release.TagName)
}

func (u *Updater) getLastRelease(ctx context.Context) (*Release, error) {
	req, err := http.NewRequest(http.MethodGet, u.releaseURL, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("call github api failed with code ")
	}

	d, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var out Release
	if err := json.Unmarshal(d, &out); err != nil {
		return nil, err
	}

	return filterAssetsForArchAndOs(out), nil
}

func filterAssetsForArchAndOs(release Release) *Release {
	r := &Release{
		TagName: release.TagName,
		Assets:  nil,
	}

	for _, a := range release.Assets {
		if strings.Contains(a.BrowserDownloadURL, runtime.GOOS) && strings.Contains(a.BrowserDownloadURL, runtime.GOARCH) {
			r.Assets = append(r.Assets, a)
		}
	}
	return r
}

func downloadAsset(ctx context.Context, url, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download asset failed, status code: %s", resp.Status)
	}

	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

func extractTarGz(source, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	src, err := os.Open(source)
	if err != nil {
		return err
	}

	uncompressedStream, err := gzip.NewReader(src)
	if err != nil {
		return err
	}

	// FIXME only extract the executable file
	tarReader := tar.NewReader(uncompressedStream)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(path.Join(dest, header.Name), 0755); err != nil {
				return fmt.Errorf("make dir failed:%w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(path.Join(dest, header.Name))
			if err != nil {
				return fmt.Errorf("create file failed:%w", err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("copy failed:%w", err)
			}
			_ = outFile.Close()

		default:
			return errors.New("unknown file structure")
		}
	}
	return nil
}
