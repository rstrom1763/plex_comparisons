package constants

import "time"

const (
	DOTENV_PATH      = ".env"
	SNAPSHOT_MAX_AGE = 24 * time.Hour

	INSTALL_DIR          = "/opt/plex_comparisons"
	INSTALLED_BINARY     = "/opt/plex_comparisons/plex_comparisons"
	DEFAULT_LOG_FILE     = "/opt/plex_comparisons/plex_comparisons.log"
	SYSTEMD_SERVICE_NAME = "plex-comparisons.service"
	SYSTEMD_SERVICE_PATH = "/etc/systemd/system/plex-comparisons.service"

	AUTH_SESSION_LENGTH_MINUTES           = 30
	AUTH_SESSION_CLEANUP_INTERVAL_MINUTES = 5
	RANDOM_TOKEN_LENGTH                   = 30
)
