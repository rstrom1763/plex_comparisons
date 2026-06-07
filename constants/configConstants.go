package constants

import "time"

const (
	DOTENV_PATH      = ".env"
	SNAPSHOT_MAX_AGE = 24 * time.Hour

	INSTALL_DIR          = "/opt/plex_comparisons"
	INSTALLED_BINARY     = "/opt/plex_comparisons/plex_comparisons"
	SYSTEMD_SERVICE_NAME = "plex-comparisons.service"
	SYSTEMD_SERVICE_PATH = "/etc/systemd/system/plex-comparisons.service"
)
