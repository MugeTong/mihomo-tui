package install

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"mihomo-tui/internal/config"
	"mihomo-tui/internal/local"
	"mihomo-tui/internal/runtimeconfig"
	"mihomo-tui/internal/subscription"
)

// Payload contains the platform-specific resources embedded by cmd/installer.
type Payload struct {
	Client        []byte
	MihomoArchive []byte
	GeoIP         []byte
	MihomoVersion string
}

// Result describes the important paths created by Install.
type Result struct {
	ClientPath string
	CorePath   string
	DataDir    string
}

// Run installs a complete per-user Mihomo TUI distribution.
func Run(payload Payload) (Result, error) {
	if err := validatePayload(payload); err != nil {
		return Result{}, err
	}
	if err := local.Prepare(); err != nil {
		return Result{}, fmt.Errorf("prepare local installation: %w", err)
	}
	binDir, err := local.BinDir()
	if err != nil {
		return Result{}, err
	}
	dataDir, err := local.DataDir()
	if err != nil {
		return Result{}, err
	}

	result := Result{
		ClientPath: filepath.Join(binDir, "mhmt"),
		CorePath:   filepath.Join(dataDir, "bin", "mihomo-v"+payload.MihomoVersion),
		DataDir:    dataDir,
	}
	for _, directory := range []string{filepath.Dir(result.CorePath), filepath.Join(dataDir, "mihomo")} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return Result{}, fmt.Errorf("create %s: %w", directory, err)
		}
	}
	if err := atomicWrite(result.ClientPath, payload.Client, 0o755); err != nil {
		return Result{}, err
	}
	if err := installGzip(result.CorePath, payload.MihomoArchive); err != nil {
		return Result{}, err
	}
	if err := atomicWrite(filepath.Join(dataDir, "mihomo", "geoip.metadb"), payload.GeoIP, 0o644); err != nil {
		return Result{}, err
	}
	if err := initializeConfigFiles(result.CorePath); err != nil {
		return Result{}, err
	}
	if err := writeSourceNotices(dataDir, payload); err != nil {
		return Result{}, err
	}
	return result, nil
}

func validatePayload(payload Payload) error {
	if len(payload.Client) == 0 || len(payload.MihomoArchive) == 0 || len(payload.GeoIP) == 0 {
		return fmt.Errorf("installer payload is incomplete")
	}
	if payload.MihomoVersion == "" {
		return fmt.Errorf("Mihomo version is required")
	}
	return nil
}

func writeSourceNotices(dataDir string, payload Payload) error {
	licenseDir := filepath.Join(dataDir, "licenses")
	mihomoSource := fmt.Sprintf("Mihomo v%s corresponding source:\nhttps://github.com/MetaCubeX/mihomo/tree/v%s\n", payload.MihomoVersion, payload.MihomoVersion)
	if err := atomicWrite(filepath.Join(licenseDir, "MIHOMO_SOURCE.txt"), []byte(mihomoSource), 0o644); err != nil {
		return err
	}
	geoSum := sha256.Sum256(payload.GeoIP)
	metaSource := fmt.Sprintf("MetaCubeX meta-rules-dat corresponding source:\nhttps://github.com/MetaCubeX/meta-rules-dat\nBundled geoip.metadb SHA-256: %x\n", geoSum)
	return atomicWrite(filepath.Join(licenseDir, "META_RULES_SOURCE.txt"), []byte(metaSource), 0o644)
}

func initializeConfigFiles(corePath string) error {
	configDir, err := local.ConfigDir()
	if err != nil {
		return err
	}
	cfg := config.Default()
	cfg.BinaryPath = corePath
	cfg.ConfigPath = filepath.Join(configDir, "config.yaml")

	settings, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode initial settings: %w", err)
	}
	if err := writeMissing(filepath.Join(configDir, "config.json"), append(settings, '\n'), 0o600); err != nil {
		return err
	}

	state := subscription.NewState()
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode initial subscription state: %w", err)
	}
	if err := writeMissing(filepath.Join(configDir, "state.json"), append(stateData, '\n'), 0o600); err != nil {
		return err
	}

	runtimeData, err := runtimeconfig.Generate(cfg, state)
	if err != nil {
		return fmt.Errorf("generate initial runtime config: %w", err)
	}
	return writeMissing(cfg.ConfigPath, runtimeData, 0o600)
}

func installGzip(destination string, archive []byte) error {
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return fmt.Errorf("open embedded Mihomo archive: %w", err)
	}
	defer reader.Close()
	return atomicWriteFrom(destination, 0o755, func(writer io.Writer) error {
		_, err := io.Copy(writer, reader)
		return err
	})
}

func writeMissing(path string, data []byte, mode fs.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	return atomicWrite(path, data, mode)
}

func atomicWrite(destination string, data []byte, mode fs.FileMode) error {
	return atomicWriteFrom(destination, mode, func(writer io.Writer) error {
		_, err := writer.Write(data)
		return err
	})
}

func atomicWriteFrom(destination string, mode fs.FileMode, write func(io.Writer) error) error {
	temporary, err := os.CreateTemp(filepath.Dir(destination), ".install-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", destination, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if err := write(temporary); err != nil {
		temporary.Close()
		return fmt.Errorf("write %s: %w", destination, err)
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return fmt.Errorf("install %s: %w", destination, err)
	}
	return nil
}
