package main

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// configDirPath returns the absolute OS specific config path for the
// application. If application directory doesn't exist, it will be
// created. If path is not found, the relativePath will be returned
// as is.
func configDirPath(relativePath string) string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		logrus.WithError(err).Warn("could not find config dir path, using relative")
	}

	appConfigPath := filepath.Join(filepath.Dir(configDir), Application)
	if err := os.MkdirAll(appConfigPath, 0755); err != nil {
		logrus.WithError(err).Warn("could not create user config dir path, using relative")
		return relativePath
	}

	return filepath.Join(appConfigPath, relativePath)
}

// binDirPath returns the absoulte path to the executable file. If
// the path is not found, the relativePath will be returned.
func binDirPath(relativePath string) string {
	execPath, err := os.Executable()
	if err != nil {
		logrus.WithError(err).Warn("could not find executable binary path, using relative")
		return relativePath
	}

	return filepath.Join(filepath.Dir(execPath), relativePath)
}
