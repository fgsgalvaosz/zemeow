package migrations

import (
	"database/sql"
	"fmt"

	"github.com/felipe/zemeow/internal/logger"
)

// IMPORTANTE: As tabelas do WhatsApp (whatsmeow_*) são criadas e gerenciadas automaticamente
// pela biblioteca go.mau.fi/whatsmeow/store/sqlstore através do método container.Upgrade().
// Não é necessário criar essas tabelas manualmente nas migrações.
//
// Este sistema de migrações é apenas para as tabelas específicas da aplicação ZeMeow,
// como a tabela 'sessions' e outras estruturas de dados da aplicação.

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
			Description: "Create sessions table and basic structure",
			Up: `
				-- Criar extensões necessárias (PostgreSQL)
				CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
				CREATE EXTENSION IF NOT EXISTS "pgcrypto";

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
			`,
			Down: `
				DROP TRIGGER IF EXISTS update_sessions_updated_at ON sessions;
				DROP FUNCTION IF EXISTS update_updated_at_column();
				DROP TABLE IF EXISTS sessions;
			`,
		},
		{
			Version:     2,
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
		{
			Version:     3,
			Description: "Create relationships between sessions and whatsmeow tables",
			Up: `
				-- Esta migração cria relacionamentos entre sessions e tabelas whatsmeow
				-- As tabelas whatsmeow são criadas automaticamente pelo container.Upgrade()

				-- Função para criar relacionamentos de forma segura
				CREATE OR REPLACE FUNCTION create_whatsmeow_relationships() RETURNS void AS $$
				BEGIN
					-- Foreign key da sessions para whatsmeow_device (se existir)
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_device') THEN
						-- Adicionar constraint apenas se não existir
						IF NOT EXISTS (
							SELECT 1 FROM information_schema.table_constraints
							WHERE constraint_name = 'fk_sessions_whatsmeow_device'
						) THEN
							ALTER TABLE sessions
							ADD CONSTRAINT fk_sessions_whatsmeow_device
							FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid)
							ON DELETE CASCADE ON UPDATE CASCADE;
						END IF;
					END IF;
				END;
				$$ LANGUAGE plpgsql;

				-- Executar a função
				SELECT create_whatsmeow_relationships();

				-- Limpar função temporária
				DROP FUNCTION IF EXISTS create_whatsmeow_relationships();
			`,
			Down: `
				-- Remover foreign key
				ALTER TABLE sessions DROP CONSTRAINT IF EXISTS fk_sessions_whatsmeow_device;
			`,
		},
		{
			Version:     4,
			Description: "Create optimized indexes for whatsmeow tables",
			Up: `
				-- Esta migração cria índices otimizados para as tabelas whatsmeow
				-- Executa apenas se as tabelas existirem

				-- Função para criar índices de forma segura
				CREATE OR REPLACE FUNCTION create_whatsmeow_indexes() RETURNS void AS $$
				BEGIN
					-- Índices para whatsmeow_device
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_device') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_device_jid_lookup ON whatsmeow_device(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_device_registration ON whatsmeow_device(registration_id);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_device_platform ON whatsmeow_device(platform) WHERE platform != '';
					END IF;

					-- Índices para whatsmeow_identity_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_identity_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_identity_our_jid ON whatsmeow_identity_keys(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_identity_their_id ON whatsmeow_identity_keys(their_id);
					END IF;

					-- Índices para whatsmeow_pre_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_pre_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_prekeys_jid ON whatsmeow_pre_keys(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_prekeys_uploaded ON whatsmeow_pre_keys(jid, uploaded);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_prekeys_key_id ON whatsmeow_pre_keys(jid, key_id);
					END IF;

					-- Índices para whatsmeow_sender_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_sender_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sender_our_jid ON whatsmeow_sender_keys(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sender_chat ON whatsmeow_sender_keys(our_jid, chat_id);
					END IF;
				END;
				$$ LANGUAGE plpgsql;

				-- Executar a função
				SELECT create_whatsmeow_indexes();

				-- Limpar função temporária
				DROP FUNCTION IF EXISTS create_whatsmeow_indexes();
			`,
			Down: `
				-- Remover índices criados
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_device_jid_lookup;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_device_registration;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_device_platform;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_identity_our_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_identity_their_id;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_prekeys_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_prekeys_uploaded;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_prekeys_key_id;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_sender_our_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_sender_chat;
			`,
		},
		{
			Version:     5,
			Description: "Create indexes for whatsmeow app state and contacts tables",
			Up: `
				-- Índices para tabelas de app state e contatos do whatsmeow

				-- Função para criar índices de app state
				CREATE OR REPLACE FUNCTION create_whatsmeow_appstate_indexes() RETURNS void AS $$
				BEGIN
					-- Índices para whatsmeow_app_state_sync_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_sync_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sync_keys_jid ON whatsmeow_app_state_sync_keys(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sync_keys_timestamp ON whatsmeow_app_state_sync_keys(jid, timestamp);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sync_keys_fingerprint ON whatsmeow_app_state_sync_keys(fingerprint);
					END IF;

					-- Índices para whatsmeow_app_state_version
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_version') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_version_jid ON whatsmeow_app_state_version(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_version_name ON whatsmeow_app_state_version(jid, name);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_version_version ON whatsmeow_app_state_version(version);
					END IF;

					-- Índices para whatsmeow_app_state_mutation_macs
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_mutation_macs') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_mutation_jid ON whatsmeow_app_state_mutation_macs(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_mutation_name ON whatsmeow_app_state_mutation_macs(jid, name);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_mutation_version ON whatsmeow_app_state_mutation_macs(jid, name, version);
					END IF;

					-- Índices para whatsmeow_contacts
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_contacts') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_contacts_our_jid ON whatsmeow_contacts(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_contacts_their_jid ON whatsmeow_contacts(their_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_contacts_names ON whatsmeow_contacts(our_jid, first_name, full_name);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_contacts_push_name ON whatsmeow_contacts(our_jid, push_name) WHERE push_name != '';
					END IF;

					-- Índices para whatsmeow_chat_settings
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_chat_settings') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_our_jid ON whatsmeow_chat_settings(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_settings ON whatsmeow_chat_settings(our_jid, chat_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_muted ON whatsmeow_chat_settings(our_jid, muted_until) WHERE muted_until > 0;
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_pinned ON whatsmeow_chat_settings(our_jid, pinned) WHERE pinned = true;
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_archived ON whatsmeow_chat_settings(our_jid, archived) WHERE archived = true;
					END IF;
				END;
				$$ LANGUAGE plpgsql;

				-- Executar a função
				SELECT create_whatsmeow_appstate_indexes();

				-- Limpar função temporária
				DROP FUNCTION IF EXISTS create_whatsmeow_appstate_indexes();
			`,
			Down: `
				-- Remover índices de app state e contatos
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_sync_keys_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_sync_keys_timestamp;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_sync_keys_fingerprint;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_version_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_version_name;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_version_version;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_mutation_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_mutation_name;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_mutation_version;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_contacts_our_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_contacts_their_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_contacts_names;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_contacts_push_name;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_chat_our_jid;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_chat_settings;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_chat_muted;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_chat_pinned;
				DROP INDEX CONCURRENTLY IF EXISTS idx_whatsmeow_chat_archived;
			`,
		},
		{
			Version:     3,
			Description: "Create relationships and indexes between sessions and whatsmeow tables",
			Up: `
				-- Aguardar que as tabelas do whatsmeow sejam criadas pelo upgrade automático
				-- Esta migração será executada após o container.Upgrade() do whatsmeow

				-- Adicionar foreign keys e relacionamentos após as tabelas existirem
				-- Função para criar relacionamentos de forma segura
				CREATE OR REPLACE FUNCTION create_whatsmeow_relationships() RETURNS void AS $$
				BEGIN
					-- Verificar se as tabelas do whatsmeow existem antes de criar relacionamentos
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_device') THEN
						-- Foreign key da sessions para whatsmeow_device
						IF NOT EXISTS (
							SELECT 1 FROM information_schema.table_constraints
							WHERE constraint_name = 'fk_sessions_whatsmeow_device'
						) THEN
							ALTER TABLE sessions
							ADD CONSTRAINT fk_sessions_whatsmeow_device
							FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid)
							ON DELETE CASCADE ON UPDATE CASCADE;
						END IF;

						-- Índices otimizados para whatsmeow_device
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_device_jid_lookup ON whatsmeow_device(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_device_registration ON whatsmeow_device(registration_id);
					END IF;

					-- Índices para whatsmeow_identity_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_identity_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_identity_our_jid ON whatsmeow_identity_keys(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_identity_their_id ON whatsmeow_identity_keys(their_id);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_identity_composite ON whatsmeow_identity_keys(our_jid, their_id);
					END IF;

					-- Índices para whatsmeow_pre_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_pre_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_prekeys_jid ON whatsmeow_pre_keys(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_prekeys_uploaded ON whatsmeow_pre_keys(jid, uploaded);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_prekeys_key_id ON whatsmeow_pre_keys(jid, key_id);
					END IF;

					-- Índices para whatsmeow_sender_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_sender_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sender_our_jid ON whatsmeow_sender_keys(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sender_chat ON whatsmeow_sender_keys(our_jid, chat_id);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sender_composite ON whatsmeow_sender_keys(our_jid, chat_id, sender_id);
					END IF;

					-- Índices para whatsmeow_app_state_sync_keys
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_sync_keys') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sync_keys_jid ON whatsmeow_app_state_sync_keys(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_sync_keys_timestamp ON whatsmeow_app_state_sync_keys(jid, timestamp);
					END IF;

					-- Índices para whatsmeow_app_state_version
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_version') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_version_jid ON whatsmeow_app_state_version(jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_version_name ON whatsmeow_app_state_version(jid, name);
					END IF;

					-- Índices para whatsmeow_contacts
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_contacts') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_contacts_our_jid ON whatsmeow_contacts(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_contacts_their_jid ON whatsmeow_contacts(their_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_contacts_names ON whatsmeow_contacts(our_jid, first_name, full_name);
					END IF;

					-- Índices para whatsmeow_chat_settings
					IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'whatsmeow_chat_settings') THEN
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_our_jid ON whatsmeow_chat_settings(our_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_settings ON whatsmeow_chat_settings(our_jid, chat_jid);
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_muted ON whatsmeow_chat_settings(our_jid, muted_until) WHERE muted_until > 0;
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_pinned ON whatsmeow_chat_settings(our_jid, pinned) WHERE pinned = true;
						CREATE INDEX IF NOT EXISTS idx_whatsmeow_chat_archived ON whatsmeow_chat_settings(our_jid, archived) WHERE archived = true;
					END IF;

				END;
				$$ LANGUAGE plpgsql;

				-- Executar a função para criar os relacionamentos
				SELECT create_whatsmeow_relationships();

				-- Remover a função após uso
				DROP FUNCTION IF EXISTS create_whatsmeow_relationships();
			`,
			Down: `
				-- Remover foreign keys
				ALTER TABLE sessions DROP CONSTRAINT IF EXISTS fk_sessions_whatsmeow_device;

				-- Remover índices criados
				DROP INDEX IF EXISTS idx_whatsmeow_device_jid_lookup;
				DROP INDEX IF EXISTS idx_whatsmeow_device_registration;
				DROP INDEX IF EXISTS idx_whatsmeow_identity_our_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_identity_their_id;
				DROP INDEX IF EXISTS idx_whatsmeow_identity_composite;
				DROP INDEX IF EXISTS idx_whatsmeow_prekeys_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_prekeys_uploaded;
				DROP INDEX IF EXISTS idx_whatsmeow_prekeys_key_id;
				DROP INDEX IF EXISTS idx_whatsmeow_sender_our_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_sender_chat;
				DROP INDEX IF EXISTS idx_whatsmeow_sender_composite;
				DROP INDEX IF EXISTS idx_whatsmeow_sync_keys_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_sync_keys_timestamp;
				DROP INDEX IF EXISTS idx_whatsmeow_version_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_version_name;
				DROP INDEX IF EXISTS idx_whatsmeow_contacts_our_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_contacts_their_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_contacts_names;
				DROP INDEX IF EXISTS idx_whatsmeow_chat_our_jid;
				DROP INDEX IF EXISTS idx_whatsmeow_chat_settings;
				DROP INDEX IF EXISTS idx_whatsmeow_chat_muted;
				DROP INDEX IF EXISTS idx_whatsmeow_chat_pinned;
				DROP INDEX IF EXISTS idx_whatsmeow_chat_archived;
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

// GetAppliedVersions retorna as versões de migração já aplicadas (método público)
func (m *Migrator) GetAppliedVersions() (map[int]bool, error) {
	return m.getAppliedVersions()
}
