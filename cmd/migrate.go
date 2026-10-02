package cmd

import (
	"fmt"

	"git.blinkpay.vn/software/authix/config"
	"git.blinkpay.vn/software/authix/internal/persistence/postgres"
	"git.blinkpay.vn/software/authix/internal/persistence/postgres/migrations"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun/migrate"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	RunE:  runMigrate,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	persister, err := postgres.NewPersister(cmd.Context(), cfg.Persistence.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	migrator := migrate.NewMigrator(persister.DB(), migrations.Migrations)

	if err := migrator.Init(cmd.Context()); err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}

	if err := migrator.Lock(cmd.Context()); err != nil {
		return err
	}
	defer migrator.Unlock(cmd.Context()) //nolint:errcheck

	group, err := migrator.Migrate(cmd.Context())
	if err != nil {
		return err
	}

	if group.IsZero() {
		log.Info().Msg("no migrations to run, database is up to date")
		return nil
	}

	log.Info().Msgf("migrated to %s", group)
	return nil
}
