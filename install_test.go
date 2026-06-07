package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInstallSystemdServiceWritesBinaryEnvAndService(t *testing.T) {
	tempDir := t.TempDir()
	sourceBinary := filepath.Join(tempDir, "new-binary")
	sourceEnv := filepath.Join(tempDir, ".env")
	installDir := filepath.Join(tempDir, "install")
	servicePath := filepath.Join(tempDir, "plex-comparisons.service")

	if err := os.WriteFile(sourceBinary, []byte("binary-v1"), 0644); err != nil {
		t.Fatalf("WriteFile(source binary) error = %v", err)
	}
	if err := os.WriteFile(sourceEnv, []byte("PORT=8080\n"), 0644); err != nil {
		t.Fatalf("WriteFile(source env) error = %v", err)
	}

	var commands []string
	cfg := testInstallConfig(installDir, servicePath, sourceBinary, sourceEnv)
	if err := installSystemdService(cfg, recordingCommandRunner(&commands)); err != nil {
		t.Fatalf("installSystemdService() error = %v", err)
	}

	if got := readTestFile(t, cfg.BinaryPath); got != "binary-v1" {
		t.Fatalf("installed binary = %q, want binary-v1", got)
	}
	if got := readTestFile(t, cfg.EnvPath); got != "PORT=8080\n" {
		t.Fatalf("installed env = %q, want copied env", got)
	}
	service := readTestFile(t, cfg.ServicePath)
	if !strings.Contains(service, "WorkingDirectory="+installDir) {
		t.Fatalf("service unit = %q, want install working directory", service)
	}
	if !strings.Contains(service, "ExecStart="+cfg.BinaryPath+" server") {
		t.Fatalf("service unit = %q, want server ExecStart", service)
	}

	wantCommands := []string{
		"systemctl daemon-reload",
		"systemctl enable plex-comparisons.service",
		"systemctl restart plex-comparisons.service",
	}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("commands = %v, want %v", commands, wantCommands)
	}
}

func TestInstallSystemdServiceIsIdempotentAndPreservesEnv(t *testing.T) {
	tempDir := t.TempDir()
	sourceBinary := filepath.Join(tempDir, "new-binary")
	sourceEnv := filepath.Join(tempDir, ".env")
	installDir := filepath.Join(tempDir, "install")
	servicePath := filepath.Join(tempDir, "plex-comparisons.service")

	if err := os.WriteFile(sourceBinary, []byte("binary-v1"), 0644); err != nil {
		t.Fatalf("WriteFile(source binary) error = %v", err)
	}
	if err := os.WriteFile(sourceEnv, []byte("PORT=8080\n"), 0644); err != nil {
		t.Fatalf("WriteFile(source env) error = %v", err)
	}

	cfg := testInstallConfig(installDir, servicePath, sourceBinary, sourceEnv)
	var commands []string
	if err := installSystemdService(cfg, recordingCommandRunner(&commands)); err != nil {
		t.Fatalf("first installSystemdService() error = %v", err)
	}

	if err := os.WriteFile(sourceBinary, []byte("binary-v2"), 0644); err != nil {
		t.Fatalf("WriteFile(source binary v2) error = %v", err)
	}
	if err := os.WriteFile(sourceEnv, []byte("PORT=9090\nPROTOCOL=http\n"), 0644); err != nil {
		t.Fatalf("WriteFile(source env v2) error = %v", err)
	}
	if err := os.WriteFile(cfg.EnvPath, []byte("PORT=7777\n"), 0644); err != nil {
		t.Fatalf("WriteFile(installed env) error = %v", err)
	}

	if err := installSystemdService(cfg, recordingCommandRunner(&commands)); err != nil {
		t.Fatalf("second installSystemdService() error = %v", err)
	}

	if got := readTestFile(t, cfg.BinaryPath); got != "binary-v2" {
		t.Fatalf("installed binary after update = %q, want binary-v2", got)
	}
	if got := readTestFile(t, cfg.EnvPath); got != "PORT=7777\nPROTOCOL=http\n" {
		t.Fatalf("installed env after update = %q, want preserved existing keys plus new defaults", got)
	}
}

func TestInstallSystemdServiceBacksUpExistingDatabaseBeforeUpdate(t *testing.T) {
	tempDir := t.TempDir()
	sourceBinary := filepath.Join(tempDir, "new-binary")
	sourceEnv := filepath.Join(tempDir, ".env")
	installDir := filepath.Join(tempDir, "install")
	servicePath := filepath.Join(tempDir, "plex-comparisons.service")

	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("MkdirAll(installDir) error = %v", err)
	}
	if err := os.WriteFile(sourceBinary, []byte("binary-v2"), 0644); err != nil {
		t.Fatalf("WriteFile(source binary) error = %v", err)
	}
	if err := os.WriteFile(sourceEnv, []byte("PORT=8080\n"), 0644); err != nil {
		t.Fatalf("WriteFile(source env) error = %v", err)
	}

	cfg := testInstallConfig(installDir, servicePath, sourceBinary, sourceEnv)
	if err := os.WriteFile(cfg.BinaryPath, []byte("binary-v1"), 0755); err != nil {
		t.Fatalf("WriteFile(installed binary) error = %v", err)
	}
	if err := os.WriteFile(cfg.EnvPath, []byte("LOCAL_DB_PATH=custom_state.db\n"), 0644); err != nil {
		t.Fatalf("WriteFile(installed env) error = %v", err)
	}
	dbPath := filepath.Join(installDir, "custom_state.db")
	if err := os.WriteFile(dbPath, []byte("db-before-update"), 0644); err != nil {
		t.Fatalf("WriteFile(db) error = %v", err)
	}

	var commands []string
	if err := installSystemdService(cfg, recordingCommandRunner(&commands)); err != nil {
		t.Fatalf("installSystemdService() error = %v", err)
	}

	backups, err := filepath.Glob(filepath.Join(installDir, "backups", "custom_state.db.*.backup"))
	if err != nil {
		t.Fatalf("Glob(backups) error = %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups = %v, want exactly one database backup", backups)
	}
	if got := readTestFile(t, backups[0]); got != "db-before-update" {
		t.Fatalf("backup content = %q, want original database content", got)
	}
	if got := readTestFile(t, cfg.BinaryPath); got != "binary-v2" {
		t.Fatalf("installed binary = %q, want update after backup", got)
	}
}

func TestInstallSystemdServiceStopsBeforeUpdateWhenBackupFails(t *testing.T) {
	tempDir := t.TempDir()
	sourceBinary := filepath.Join(tempDir, "new-binary")
	sourceEnv := filepath.Join(tempDir, ".env")
	installDir := filepath.Join(tempDir, "install")
	servicePath := filepath.Join(tempDir, "plex-comparisons.service")

	if err := os.MkdirAll(installDir, 0755); err != nil {
		t.Fatalf("MkdirAll(installDir) error = %v", err)
	}
	if err := os.WriteFile(sourceBinary, []byte("binary-v2"), 0644); err != nil {
		t.Fatalf("WriteFile(source binary) error = %v", err)
	}
	if err := os.WriteFile(sourceEnv, []byte("PORT=8080\n"), 0644); err != nil {
		t.Fatalf("WriteFile(source env) error = %v", err)
	}

	cfg := testInstallConfig(installDir, servicePath, sourceBinary, sourceEnv)
	if err := os.WriteFile(cfg.BinaryPath, []byte("binary-v1"), 0755); err != nil {
		t.Fatalf("WriteFile(installed binary) error = %v", err)
	}
	if err := os.WriteFile(cfg.EnvPath, []byte("LOCAL_DB_PATH=bad-db-path\n"), 0644); err != nil {
		t.Fatalf("WriteFile(installed env) error = %v", err)
	}
	if err := os.Mkdir(filepath.Join(installDir, "bad-db-path"), 0755); err != nil {
		t.Fatalf("Mkdir(db path) error = %v", err)
	}

	var commands []string
	if err := installSystemdService(cfg, recordingCommandRunner(&commands)); err == nil {
		t.Fatal("installSystemdService() error = nil, want backup failure")
	}

	if got := readTestFile(t, cfg.BinaryPath); got != "binary-v1" {
		t.Fatalf("installed binary = %q, want unchanged binary after backup failure", got)
	}
	if len(commands) != 0 {
		t.Fatalf("commands = %v, want no systemctl commands after backup failure", commands)
	}
}

func testInstallConfig(installDir string, servicePath string, sourceBinary string, sourceEnv string) installConfig {
	return installConfig{
		InstallDir:       installDir,
		BinaryPath:       filepath.Join(installDir, "plex_comparisons"),
		EnvPath:          filepath.Join(installDir, ".env"),
		ServiceName:      "plex-comparisons.service",
		ServicePath:      servicePath,
		SourceBinary:     sourceBinary,
		SourceEnv:        sourceEnv,
		SystemctlCommand: "systemctl",
	}
}

func recordingCommandRunner(commands *[]string) commandRunner {
	return func(name string, args ...string) error {
		*commands = append(*commands, strings.Join(append([]string{name}, args...), " "))
		return nil
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}
