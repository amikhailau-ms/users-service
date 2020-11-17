package db

import (
	"net/url"

	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/database/postgres" // import postgres driver
	_ "github.com/golang-migrate/migrate/source/file"       // source file driver
	log "github.com/sirupsen/logrus"
)

func Migrate(u *url.URL, src *url.URL) error {
	logger := log.WithFields(log.Fields{
		"db":     u.Hostname(),
		"schema": src.String(),
	})

	logger.Info("migrating db")
	m, err := migrate.New(
		src.String(),
		u.String(),
	)
	if err != nil {
		return err
	}

	logVersion(logger, m, "initial db schema version")
	newErr := m.Up()
	logVersion(logger, m, "current db schema version")
	return newErr
}

func logVersion(logger *log.Entry, m *migrate.Migrate, desc string) {
	version, dirty, err := m.Version()
	if err != nil {
		logger.WithError(err).Warnf("failed to get %s", desc)
	}
	logger.WithFields(log.Fields{
		"version": version,
		"dirty":   dirty,
	}).Warn(desc)
}
