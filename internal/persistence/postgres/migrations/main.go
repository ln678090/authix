package migrations

import (
	"embed"

	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun/migrate"
)

//go:embed *.sql
var sqlMigrations embed.FS

var Migrations = migrate.NewMigrations()

func init() {
	if err := Migrations.Discover(sqlMigrations); err != nil {
		log.Fatal().Err(err).Msg("failed to discover migrations")
	}
}
