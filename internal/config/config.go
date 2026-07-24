// Package config provides configuration management utilities for the 3x-ui panel,
// including version information, logging levels, database paths, and environment variable handling.
package config

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

//go:embed version
var version string

//go:embed name
var name string

// buildCommit and buildDate are injected at build time via `-ldflags -X` for
// CI per-commit (dev channel) builds; see .github/workflows/release.yml. They
// stay empty for a plain `go build` and for stable tagged releases, which is how
// IsDevBuild tells a rolling dev build apart from a stable/local one.
var (
	buildCommit string
	buildDate   string
)

// LogLevel represents the logging level for the application.
type LogLevel string

// Logging level constants
const (
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Notice  LogLevel = "notice"
	Warning LogLevel = "warning"
	Error   LogLevel = "error"
)

// GetBaseVersion returns the raw embedded release version of the 3x-ui panel
// (e.g. "3.4.0"). This is the panel's own version, not the Xray version. For the
// version a panel advertises/displays (which adds a "dev+<sha>" label on dev
// builds), use GetPanelVersion.
func GetBaseVersion() string {
	return strings.TrimSpace(version)
}

// GetName returns the name of the 3x-ui application.
func GetName() string {
	return strings.TrimSpace(name)
}

// GetBuildCommit returns the short git commit this binary was built from, or an
// empty string for a plain/local build or a stable tagged release.
func GetBuildCommit() string {
	return strings.TrimSpace(buildCommit)
}

// GetBuildDate returns the UTC build timestamp injected at build time, or empty.
func GetBuildDate() string {
	return strings.TrimSpace(buildDate)
}

// IsDevBuild reports whether this binary is a CI per-commit (dev channel) build,
// detected by the injected commit. Stable releases and local builds return false.
func IsDevBuild() bool {
	return GetBuildCommit() != ""
}

// GetPanelVersion returns the version a panel advertises to a managing master
// node and displays in the UI: the plain version for stable builds, or
// "dev+<short commit>" for dev builds. The dev form mirrors the master's
// getPanelUpdateInfo latestVersion so a node on the current dev commit compares
// as up to date instead of always showing "update available".
func GetPanelVersion() string {
	if !IsDevBuild() {
		return GetBaseVersion()
	}
	commit := GetBuildCommit()
	if len(commit) > 8 {
		commit = commit[:8]
	}
	return "dev+" + commit
}

// GetLogLevel returns the current logging level based on environment variables or defaults to Info.
func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("XUI_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

// IsDebug returns true if debug mode is enabled via the XUI_DEBUG environment variable.
func IsDebug() bool {
	return os.Getenv("XUI_DEBUG") == "true"
}

// IsSkipHSTS returns true if skipping HSTS mode is enabled via the XUI_SKIP_HSTS environment variable.
func IsSkipHSTS() bool {
	return os.Getenv("XUI_SKIP_HSTS") == "true"
}

// IsBehindHTTPSProxy marks deployments where TLS terminates at the local
// reverse proxy. It lets the application issue Secure cookies and HSTS even
// though its loopback listener itself is plain HTTP.
func IsBehindHTTPSProxy() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("XUI_BEHIND_HTTPS_PROXY")), "true")
}

// IsCommercialProduction reports whether the hardened commercial deployment
// profile is active. Keep this check in one place so security-sensitive
// behavior (cookies, branding and update policy) cannot drift between
// packages.
func IsCommercialProduction() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("XUI_COMMERCIAL_ENV")), "production")
}

func GetPortOverride() (port int, configured bool, err error) {
	value, ok := os.LookupEnv("XUI_PORT")
	if !ok || strings.TrimSpace(value) == "" {
		return 0, false, nil
	}

	port, err = strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, true, fmt.Errorf("parse XUI_PORT: %w", err)
	}
	if port < 1 || port > 65535 {
		return 0, true, fmt.Errorf("XUI_PORT must be between 1 and 65535")
	}

	return port, true, nil
}

// GetAdminBasePath returns the externally visible administrator route. The
// commercial production profile requires an independently configured,
// unguessable URL-safe path. New deployments use 40 cryptographically random
// characters; the former 18-digit form remains readable only so an existing
// installation can rotate without being locked out.
func GetAdminBasePath(fallback string) (string, error) {
	raw := strings.TrimSpace(os.Getenv("XUI_ADMIN_BASE_PATH"))
	if raw == "" {
		if IsCommercialProduction() {
			return "", fmt.Errorf("XUI_ADMIN_BASE_PATH is required in commercial production")
		}
		return normalizeURLBasePath(fallback), nil
	}

	value := strings.Trim(raw, "/")
	legacy := len(value) == 18
	strong := len(value) == 40
	if !legacy && !strong {
		return "", fmt.Errorf("XUI_ADMIN_BASE_PATH must contain a 40-character URL-safe secret")
	}
	for _, char := range []byte(value) {
		if legacy {
			if char < '0' || char > '9' {
				return "", fmt.Errorf("legacy XUI_ADMIN_BASE_PATH must contain exactly 18 ASCII digits")
			}
			continue
		}
		isLower := char >= 'a' && char <= 'z'
		isUpper := char >= 'A' && char <= 'Z'
		isDigit := char >= '0' && char <= '9'
		if !isLower && !isUpper && !isDigit && char != '-' && char != '_' {
			return "", fmt.Errorf("XUI_ADMIN_BASE_PATH must use only ASCII letters, digits, '-' and '_'")
		}
	}
	if strong {
		hasUpper, hasLower, hasDigit, hasSymbol := false, false, false, false
		for _, char := range []byte(value) {
			switch {
			case char >= 'A' && char <= 'Z':
				hasUpper = true
			case char >= 'a' && char <= 'z':
				hasLower = true
			case char >= '0' && char <= '9':
				hasDigit = true
			case char == '-' || char == '_':
				hasSymbol = true
			}
		}
		if !hasUpper || !hasLower || !hasDigit || !hasSymbol {
			return "", fmt.Errorf("XUI_ADMIN_BASE_PATH must include uppercase, lowercase, digit and '-' or '_' characters")
		}
	}
	return "/" + value + "/", nil
}

func normalizeURLBasePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return "/"
	}
	return "/" + strings.Trim(value, "/") + "/"
}

// GetBinFolderPath returns the path to the binary folder, defaulting to "bin" if not set via XUI_BIN_FOLDER.
func GetBinFolderPath() string {
	binFolderPath := os.Getenv("XUI_BIN_FOLDER")
	if binFolderPath == "" {
		binFolderPath = "bin"
	}
	return binFolderPath
}

func getBaseDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "."
	}
	exeDir := filepath.Dir(exePath)
	exeDirLower := strings.ToLower(filepath.ToSlash(exeDir))
	if strings.Contains(exeDirLower, "/appdata/local/temp/") || strings.Contains(exeDirLower, "/go-build") {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	return exeDir
}

// GetDBFolderPath returns the path to the database folder based on environment variables or platform defaults.
func GetDBFolderPath() string {
	dbFolderPath := os.Getenv("XUI_DB_FOLDER")
	if dbFolderPath != "" {
		return dbFolderPath
	}
	if runtime.GOOS == "windows" {
		return getBaseDir()
	}
	return "/etc/x-ui"
}

// GetDBPath returns the full path to the database file.
func GetDBPath() string {
	return fmt.Sprintf("%s/%s.db", GetDBFolderPath(), GetName())
}

// GetUpdateStatusFilePath returns the path to the panel self-update status
// file update.sh writes on completion. It lives beside the database, outside
// XUI_MAIN_FOLDER, so it survives an update regardless of what happens to
// that folder.
func GetUpdateStatusFilePath() string {
	return filepath.Join(GetDBFolderPath(), "update-status.json")
}

// GetDBKind returns the configured database backend: "sqlite" (default) or "postgres".
func GetDBKind() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("XUI_DB_TYPE")))
	switch v {
	case "postgres", "postgresql", "pg":
		return "postgres"
	default:
		return "sqlite"
	}
}

// GetDBDSN returns the PostgreSQL DSN from XUI_DB_DSN. Empty for sqlite.
func GetDBDSN() string {
	return strings.TrimSpace(os.Getenv("XUI_DB_DSN"))
}

// GetEnvFilePaths returns the candidate service environment file paths (the file
// systemd loads via EnvironmentFile) across the supported distro families.
func GetEnvFilePaths() []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	return []string{
		"/etc/default/x-ui",
		"/etc/conf.d/x-ui",
		"/etc/sysconfig/x-ui",
	}
}

// GetLogFolder returns the path to the log folder based on environment variables or platform defaults.
func GetLogFolder() string {
	logFolderPath := os.Getenv("XUI_LOG_FOLDER")
	if logFolderPath != "" {
		return logFolderPath
	}
	// Under `go test` the Windows default below is CWD-relative ("./log"), which
	// scatters a log/ directory through the source tree (one per tested package).
	// Redirect test runs to a shared temp folder so the source tree stays clean.
	if testing.Testing() {
		return filepath.Join(os.TempDir(), "3x-ui-test-log")
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "log")
	}
	return "/var/log/x-ui"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Sync()
}

func init() {
	if runtime.GOOS != "windows" {
		return
	}
	if os.Getenv("XUI_DB_FOLDER") != "" {
		return
	}
	oldDBFolder := "/etc/x-ui"
	oldDBPath := fmt.Sprintf("%s/%s.db", oldDBFolder, GetName())
	newDBFolder := GetDBFolderPath()
	newDBPath := fmt.Sprintf("%s/%s.db", newDBFolder, GetName())
	_, err := os.Stat(newDBPath)
	if err == nil {
		return // new exists
	}
	_, err = os.Stat(oldDBPath)
	if os.IsNotExist(err) {
		return // old does not exist
	}
	_ = copyFile(oldDBPath, newDBPath) // ignore error
}
