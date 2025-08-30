package migrations

import (
	"database/sql"
	"fmt"

	"github.com/felipe/zemeow/internal/logger"
)

// Migration representa uma migração do banco de dados
type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

// Migrator gerencia as migrações do banco de dados
type Migrator struct {
	db     *sql.DB
	logger logger.Logger
}

// NewMigrator cria um novo migrator
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger.Get(),
	}
}

// GetMigrations retorna todas as migrações disponíveis
func (m *Migrator) GetMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Create sessions table with whatsmeow store tables",
			Up: `
				-- Criar tabela sessions
				CREATE TABLE IF NOT EXISTS sessions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					session_id VARCHAR(255) UNIQUE NOT NULL,
					name VARCHAR(255) NOT NULL,
					api_key VARCHAR(255) UNIQUE NOT NULL,
					jid VARCHAR(255),
					status VARCHAR(50) DEFAULT 'disconnected',
					proxy_enabled BOOLEAN DEFAULT FALSE,
					proxy_host VARCHAR(255),
					proxy_port INTEGER,
					proxy_username VARCHAR(255),
					proxy_password VARCHAR(255),
					webhook_url VARCHAR(500),
					webhook_events TEXT[],
					created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
					updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
					last_connected_at TIMESTAMP WITH TIME ZONE,
					metadata JSONB DEFAULT '{}'
				);
				
				-- Índices para performance
				CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);
				CREATE INDEX IF NOT EXISTS idx_sessions_api_key ON sessions(api_key);
				CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
				CREATE INDEX IF NOT EXISTS idx_sessions_jid ON sessions(jid);
				CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);
				
				-- Função para atualizar updated_at automaticamente
				CREATE OR REPLACE FUNCTION update_updated_at_column()
				RETURNS TRIGGER AS $$
				BEGIN
					NEW.updated_at = NOW();
					RETURN NEW;
				END;
				$$ language 'plpgsql';
				
				-- Trigger para atualizar updated_at
				DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
				CREATE TRIGGER update_sessions_updated_at 
					BEFORE UPDATE ON sessions 
					FOR EACH ROW 
					EXECUTE FUNCTION update_updated_at_column();
				
				-- Tabelas do whatsmeow store (integração com go.mau.fi/whatsmeow/store/sqlstore)
				CREATE TABLE IF NOT EXISTS whatsmeow_device (
					jid TEXT PRIMARY KEY,
					registration_id BIGINT NOT NULL CHECK (registration_id >= 0 AND registration_id < 4294967296),
					noise_key BYTEA NOT NULL CHECK (length(noise_key) = 32),
					identity_key BYTEA NOT NULL CHECK (length(identity_key) = 32),
					signed_pre_key BYTEA NOT NULL,
					signed_pre_key_id INTEGER NOT NULL CHECK (signed_pre_key_id >= 0 AND signed_pre_key_id < 16777216),
					signed_pre_key_sig BYTEA NOT NULL CHECK (length(signed_pre_key_sig) = 64),
					adv_key BYTEA NOT NULL,
					adv_details BYTEA NOT NULL,
					adv_account_sig BYTEA NOT NULL,
					adv_account_sig_key BYTEA NOT NULL,
					adv_device_sig BYTEA NOT NULL,
					platform TEXT NOT NULL DEFAULT '',
					business_name TEXT NOT NULL DEFAULT '',
					push_name TEXT NOT NULL DEFAULT ''
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_identity_keys (
					our_jid TEXT,
					their_id TEXT,
					identity BYTEA NOT NULL,
					PRIMARY KEY (our_jid, their_id)
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_pre_keys (
					jid TEXT,
					key_id INTEGER,
					key BYTEA NOT NULL,
					uploaded BOOLEAN NOT NULL,
					PRIMARY KEY (jid, key_id)
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_sender_keys (
					our_jid TEXT,
					chat_id TEXT,
					sender_id TEXT,
					sender_key BYTEA NOT NULL,
					PRIMARY KEY (our_jid, chat_id, sender_id)
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_app_state_sync_keys (
					jid TEXT,
					key_id BYTEA,
					key_data BYTEA NOT NULL,
					timestamp BIGINT NOT NULL,
					fingerprint BYTEA NOT NULL,
					PRIMARY KEY (jid, key_id)
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_app_state_version (
					jid TEXT,
					name TEXT,
					version BIGINT NOT NULL,
					hash BYTEA NOT NULL,
					PRIMARY KEY (jid, name)
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_app_state_mutation_macs (
					jid TEXT,
					name TEXT,
					version BIGINT,
					index_mac BYTEA,
					value_mac BYTEA,
					PRIMARY KEY (jid, name, version, index_mac)
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
					our_jid TEXT,
					their_jid TEXT,
					first_name TEXT,
					full_name TEXT,
					push_name TEXT,
					business_name TEXT,
					PRIMARY KEY (our_jid, their_jid)
				);
				
				CREATE TABLE IF NOT EXISTS whatsmeow_chat_settings (
					our_jid TEXT,
					chat_jid TEXT,
					muted_until BIGINT NOT NULL DEFAULT 0,
					pinned BOOLEAN NOT NULL DEFAULT false,
					archived BOOLEAN NOT NULL DEFAULT false,
					PRIMARY KEY (our_jid, chat_jid)
				);
			`,
			Down: `
				DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
				DROP FUNCTION IF EXISTS update_updated_at_column();
				DROP TABLE IF EXISTS whatsmeow_chat_settings;
				DROP TABLE IF EXISTS whatsmeow_contacts;
				DROP TABLE IF EXISTS whatsmeow_app_state_mutation_macs;
				DROP TABLE IF EXISTS whatsmeow_app_state_version;
				DROP TABLE IF EXISTS whatsmeow_app_state_sync_keys;
				DROP TABLE IF EXISTS whatsmeow_sender_keys;
				DROP TABLE IF EXISTS whatsmeow_pre_keys;
				DROP TABLE IF EXISTS whatsmeow_identity_keys;
				DROP TABLE IF EXISTS whatsmeow_device;
				DROP TABLE IF EXISTS sessions;
			`,
		},
		{
			Version:     2,
			Description: "Create migrations table",
			Up: `
				-- Tabela para controlar migrações
				CREATE TABLE IF NOT EXISTS schema_migrations (
					version INTEGER PRIMARY KEY,
					description TEXT NOT NULL,
					applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
				);
			`,
			Down: `
				DROP TABLE IF EXISTS schema_migrations;
			`,
		},
		{
			Version:     3,
			Description: "Add session statistics columns",
			Up: `
				-- Adicionar colunas de estatísticas
				ALTER TABLE sessions 
				ADD COLUMN IF NOT EXISTS messages_received INTEGER DEFAULT 0,
				ADD COLUMN IF NOT EXISTS messages_sent INTEGER DEFAULT 0,
				ADD COLUMN IF NOT EXISTS reconnections INTEGER DEFAULT 0,
				ADD COLUMN IF NOT EXISTS last_activity TIMESTAMP WITH TIME ZONE;
				
				-- Índice para last_activity
				CREATE INDEX IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity);
			`,
			Down: `
				DROP INDEX IF EXISTS idx_sessions_last_activity;
				ALTER TABLE sessions 
				DROP COLUMN IF EXISTS messages_received,
				DROP COLUMN IF EXISTS messages_sent,
				DROP COLUMN IF EXISTS reconnections,
				DROP COLUMN IF EXISTS last_activity;
			`,
		},
	}
}

// Run executa todas as migrações pendentes
func (m *Migrator) Run() error {
	m.logger.Info().Msg("Starting database migrations")

	// Criar tabela de migrações se não existir
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

// createMigrationsTable cria a tabela de controle de migrações
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

// getAppliedVersions retorna as versões já aplicadas
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

// isApplied verifica se uma migração já foi aplicada
func (m *Migrator) isApplied(version int, applied map[int]bool) bool {
	return applied[version]
}

// applyMigration aplica uma migração específica
func (m *Migrator) applyMigration(migration Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Executar a migração
	if _, err := tx.Exec(migration.Up); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Registrar a migração como aplicada
	if _, err := tx.Exec(
		"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
		migration.Version, migration.Description,
	); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// Rollback desfaz uma migração específica
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

	// Executar rollback
	if _, err := tx.Exec(targetMigration.Down); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remover registro da migração
	if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	m.logger.Info().Int("version", version).Msg("Migration rolled back successfully")
	return nil
}

// Status retorna o status das migrações
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

// MigrationStatus representa o status de uma migração
type MigrationStatus struct {
	Version     int    `json:"version"`
	Description string `json:"description"`
	Applied     bool   `json:"applied"`
}
