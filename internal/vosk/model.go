package vosk

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

var logger zerolog.Logger

const (
	DefaultModel = "vosk-model-en-us-0.22"
)

func getModel(log zerolog.Logger) (string, error) {
	logger = log.With().Str("component", "model_downloader").Logger()
	modelPath := filepath.Join("models", DefaultModel)

	if _, err := os.Stat(modelPath); err == nil {
		logger.Info().Str("path", modelPath).Msg("Model already exists, skipping download")
		return modelPath, nil
	}

	downloadUrl := fmt.Sprintf("https://alphacephei.com/vosk/models/%s.zip", DefaultModel)
	return modelPath, downloadAndUnzip(downloadUrl, "models")
}

func downloadAndUnzip(url, destDir string) error {
	logger.Info().Str("url", url).Msg("Downloading and unzipping vosk model")
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download url '%s': %w", url, err)
	}
	defer resp.Body.Close()

	tmpFile, err := os.CreateTemp("", "download-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	logger.Debug().Str("tempFile", tmpFile.Name()).Msg("Writing downloaded content to temp file")
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	return unzip(tmpFile.Name(), destDir)
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer r.Close()

	logger.Info().Str("src", src).Str("dest", dest).Msg("Unzipping model")
	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to open file for writing: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed to copy file contents: %w", err)
		}
	}

	logger.Info().Msg("Unzipping completed successfully")
	return nil
}
