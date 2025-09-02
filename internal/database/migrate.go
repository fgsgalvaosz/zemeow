package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/felipe/zemeow/internal/logger"
)

type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

type Migrator struct {
	db     *sql.DB
	logger logger.Logger
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger.Get(),
	}
}

func (m *Migrator) GetMigrations() []Migration {
	migrations, err := m.loadMigrationsFromFiles()
	if err != nil {
		m.logger.Error().Err(err).Msg("Failed to load migrations from files, falling back to embedded migrations")
		return m.getFallbackMigrations()
	}
	return migrations
}

func (m *Migrator) loadMigrationsFromFiles() ([]Migration, error) {
	migrationsDir := "internal/db/migrations"

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*_up.sql"))
	if err != nil {
		return nil, fmt.Errorf("failed to find migration files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no migration files found in %s", migrationsDir)
	}

	var migrations []Migration
	for _, upFile := range files {

		basename := filepath.Base(upFile)
		parts := strings.Split(basename, "_")
		if len(parts) < 2 {
			continue
		}

		versionStr := parts[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			m.logger.Warn().Str("file", basename).Msg("Invalid migration version number")
			continue
		}

		upSQL, err := os.ReadFile(upFile)
		if err != nil {
			m.logger.Error().Err(err).Str("file", upFile).Msg("Failed to read up migration file")
			continue
		}

		downFile := strings.Replace(upFile, "_up.sql", "_down.sql", 1)
		downSQL, err := os.ReadFile(downFile)
		if err != nil {
			m.logger.Error().Err(err).Str("file", downFile).Msg("Failed to read down migration file")
			continue
		}

		description := strings.Join(parts[1:], "_")
		description = strings.TrimSuffix(description, "_up.sql")
		description = strings.ReplaceAll(description, "_", " ")

		migrations = append(migrations, Migration{
			Version:     version,
			Description: description,
			Up:          string(upSQL),
			Down:        string(downSQL),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	m.logger.Info().Int("count", len(migrations)).Msg("Loaded migrations from SQL files")
	return migrations, nil
}

func (m *Migrator) getFallbackMigrations() []Migration {
	m.logger.Warn().Msg("No SQL migration files found - please ensure migration files exist in internal/db/migrations/")
	return []Migration{}
}

func (m *Migrator) Run() error {
	m.logger.Info().Msg("Starting database migrations")

	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	migrations := m.GetMigrations()
	appliedVersions, err := m.getAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	for _, migration := range migrations {
		if m.isApplied(migration.Version, appliedVersions) {
			m.logger.Debug().Int("version", migration.Version).Msg("Migration already applied")
			continue
		}

		m.logger.Info().Int("version", migration.Version).Str("description", migration.Description).Msg("Applying migration")

		if err := m.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		m.logger.Info().Int("version", migration.Version).Msg("Migration applied successfully")
	}

	m.logger.Info().Msg("All migrations completed successfully")
	return nil
}

func (m *Migrator) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) getAppliedVersions() (map[int]bool, error) {
	query := "SELECT version FROM schema_migrations"
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func (m *Migrator) isApplied(version int, applied map[int]bool) bool {
	return applied[version]
}

func (m *Migrator) applyMigration(migration Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(migration.Up); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
		migration.Version, migration.Description,
	); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

func (m *Migrator) Rollback(version int) error {
	migrations := m.GetMigrations()
	var targetMigration *Migration

	for _, migration := range migrations {
		if migration.Version == version {
			targetMigration = &migration
			break
		}
	}

	if targetMigration == nil {
		return fmt.Errorf("migration version %d not found", version)
	}

	m.logger.Info().Int("version", version).Str("description", targetMigration.Description).Msg("Rolling back migration")

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(targetMigration.Down); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	m.logger.Info().Int("version", version).Msg("Migration rolled back successfully")
	return nil
}

func (m *Migrator) Status() ([]MigrationStatus, error) {
	migrations := m.GetMigrations()
	appliedVersions, err := m.getAppliedVersions()
	if err != nil {
		return nil, err
	}

	status := make([]MigrationStatus, len(migrations))
	for i, migration := range migrations {
		status[i] = MigrationStatus{
			Version:     migration.Version,
			Description: migration.Description,
			Applied:     m.isApplied(migration.Version, appliedVersions),
		}
	}

	return status, nil
}

type MigrationStatus struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
}

func (m *Migrator) GetAppliedVersions() (map[int]bool, error) {
	return m.getAppliedVersions()
}
