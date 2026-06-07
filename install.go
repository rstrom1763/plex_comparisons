package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rstrom1763/plex_comparisons/constants"
)

type installConfig struct {
	InstallDir       string
	BinaryPath       string
	EnvPath          string
	ServiceName      string
	ServicePath      string
	SourceBinary     string
	SourceEnv        string
	SystemctlCommand string
}

type commandRunner func(name string, args ...string) error

func install() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("install is only supported on Linux systems using systemd")
	}

	sourceBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate running binary: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current working directory: %w", err)
	}

	cfg := defaultInstallConfig(sourceBinary, filepath.Join(cwd, constants.DOTENV_PATH))
	return installSystemdService(cfg, runCommand)
}

func defaultInstallConfig(sourceBinary string, sourceEnv string) installConfig {
	return installConfig{
		InstallDir:       constants.INSTALL_DIR,
		BinaryPath:       constants.INSTALLED_BINARY,
		EnvPath:          filepath.Join(constants.INSTALL_DIR, constants.DOTENV_PATH),
		ServiceName:      constants.SYSTEMD_SERVICE_NAME,
		ServicePath:      constants.SYSTEMD_SERVICE_PATH,
		SourceBinary:     sourceBinary,
		SourceEnv:        sourceEnv,
		SystemctlCommand: "systemctl",
	}
}

func installSystemdService(cfg installConfig, run commandRunner) error {
	if err := backupExistingInstallDatabase(cfg); err != nil {
		return fmt.Errorf("could not backup existing database: %w", err)
	}

	if err := os.MkdirAll(cfg.InstallDir, 0755); err != nil {
		return fmt.Errorf("could not create install directory: %w", err)
	}

	if err := copyFileIfChanged(cfg.SourceBinary, cfg.BinaryPath, 0755); err != nil {
		return fmt.Errorf("could not install binary: %w", err)
	}

	if err := mergeEnvFile(cfg.SourceEnv, cfg.EnvPath, 0644); err != nil {
		return fmt.Errorf("could not install env file: %w", err)
	}
	if err := ensureEnvValue(cfg.EnvPath, "LOG_FILE", defaultLogFilePath(cfg), 0644); err != nil {
		return fmt.Errorf("could not configure log file: %w", err)
	}

	unit := systemdServiceUnit(cfg)
	if err := writeFileIfChanged(cfg.ServicePath, []byte(unit), 0644); err != nil {
		return fmt.Errorf("could not write systemd service: %w", err)
	}

	if err := run(cfg.SystemctlCommand, "daemon-reload"); err != nil {
		return fmt.Errorf("could not reload systemd: %w", err)
	}
	if err := run(cfg.SystemctlCommand, "enable", cfg.ServiceName); err != nil {
		return fmt.Errorf("could not enable systemd service: %w", err)
	}
	if err := run(cfg.SystemctlCommand, "restart", cfg.ServiceName); err != nil {
		return fmt.Errorf("could not restart systemd service: %w", err)
	}

	return nil
}

func systemdServiceUnit(cfg installConfig) string {
	return fmt.Sprintf(`[Unit]
Description=Plex Comparisons
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s server
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, cfg.InstallDir, cfg.BinaryPath)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func backupExistingInstallDatabase(cfg installConfig) error {
	if _, err := os.Stat(cfg.InstallDir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	dbPath, err := installedDatabasePath(cfg)
	if err != nil {
		return err
	}

	info, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("database path is a directory: %s", dbPath)
	}

	backupDir := filepath.Join(cfg.InstallDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	backupPath := filepath.Join(backupDir, fmt.Sprintf("%s.%s.backup", filepath.Base(dbPath), time.Now().UTC().Format("20060102T150405.000000000Z")))
	return copyFileIfChanged(dbPath, backupPath, info.Mode().Perm())
}

func installedDatabasePath(cfg installConfig) (string, error) {
	envData, err := os.ReadFile(cfg.EnvPath)
	if os.IsNotExist(err) {
		return filepath.Join(cfg.InstallDir, "local_state.db"), nil
	}
	if err != nil {
		return "", err
	}

	if dbPath, ok := envValue(envData, "LOCAL_DB_PATH"); ok && dbPath != "" {
		if filepath.IsAbs(dbPath) {
			return dbPath, nil
		}
		return filepath.Join(cfg.InstallDir, dbPath), nil
	}
	return filepath.Join(cfg.InstallDir, "local_state.db"), nil
}

func copyFileIfMissing(src string, dst string, mode os.FileMode) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	return copyFileIfChanged(src, dst, mode)
}

func mergeEnvFile(src string, dst string, mode os.FileMode) error {
	srcData, err := os.ReadFile(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	dstData, err := os.ReadFile(dst)
	if os.IsNotExist(err) {
		return writeFileIfChanged(dst, srcData, mode)
	}
	if err != nil {
		return err
	}

	existingKeys := envKeys(dstData)
	var merged bytes.Buffer
	merged.Write(dstData)

	for _, line := range strings.SplitAfter(string(srcData), "\n") {
		key, ok := envKey(line)
		if !ok || existingKeys[key] {
			continue
		}
		if merged.Len() > 0 && !strings.HasSuffix(merged.String(), "\n") {
			merged.WriteByte('\n')
		}
		merged.WriteString(line)
		if !strings.HasSuffix(line, "\n") {
			merged.WriteByte('\n')
		}
		existingKeys[key] = true
	}

	return writeFileIfChanged(dst, merged.Bytes(), mode)
}

func ensureEnvValue(path string, key string, value string, mode os.FileMode) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return writeFileIfChanged(path, []byte(fmt.Sprintf("%s=%s\n", key, value)), mode)
	}
	if err != nil {
		return err
	}
	if _, ok := envValue(data, key); ok {
		return os.Chmod(path, mode)
	}

	var updated bytes.Buffer
	updated.Write(data)
	if updated.Len() > 0 && !strings.HasSuffix(updated.String(), "\n") {
		updated.WriteByte('\n')
	}
	updated.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	return writeFileIfChanged(path, updated.Bytes(), mode)
}

func defaultLogFilePath(cfg installConfig) string {
	return filepath.Join(cfg.InstallDir, filepath.Base(constants.DEFAULT_LOG_FILE))
}

func envKeys(data []byte) map[string]bool {
	keys := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		if key, ok := envKey(line); ok {
			keys[key] = true
		}
	}
	return keys
}

func envValue(data []byte, targetKey string) (string, bool) {
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := envEntry(line)
		if ok && key == targetKey {
			return value, true
		}
	}
	return "", false
}

func envKey(line string) (string, bool) {
	key, _, ok := envEntry(line)
	return key, ok
}

func envEntry(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")
	idx := strings.Index(line, "=")
	if idx <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	if key == "" {
		return "", "", false
	}
	value := strings.TrimSpace(line[idx+1:])
	value = strings.Trim(value, `"'`)
	return key, value, true
}

func copyFileIfChanged(src string, dst string, mode os.FileMode) error {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	if dstData, err := os.ReadFile(dst); err == nil && bytes.Equal(srcData, dstData) {
		return os.Chmod(dst, mode)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	return writeFileIfChanged(dst, srcData, mode)
}

func writeFileIfChanged(path string, data []byte, mode os.FileMode) error {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return os.Chmod(path, mode)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	if _, err := io.Copy(tempFile, bytes.NewReader(data)); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		_ = os.Remove(path)
	}
	return os.Rename(tempPath, path)
}
