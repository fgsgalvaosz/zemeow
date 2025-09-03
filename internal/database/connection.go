package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/logger"
	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

type DB struct {
	*sql.DB
	config *config.DatabaseConfig
	logger *logger.ComponentLogger
}

func Connect(cfg *config.Config) (*DB, error) {
	return New(&cfg.Database)
}

func Migrate(db *DB) error {
	return db.Migrate()
}

func New(cfg *config.DatabaseConfig) (*DB, error) {
	compLogger := logger.ForComponent("database")
	connectOp := compLogger.ForOperation("connect")

	connectOp.Starting().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Name).
		Msg(logger.GetStandardizedMessage("database", "connect", "starting"))

	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	connectOp.Success().
		Msg(logger.GetStandardizedMessage("database", "connect", "success"))

	return &DB{
		DB:     db,
		config: cfg,
		logger: compLogger,
	}, nil
}

func (db *DB) Close() error {
	db.logger.Info().Msg("Closing database connection")
	return db.DB.Close()
}

func (db *DB) GetSQLStore() *sqlstore.Container {
	storeOp := db.logger.ForOperation("create_sqlstore")

	storeOp.Starting().
		Msg(logger.GetStandardizedMessage("database", "create_sqlstore", "starting"))

	whatsmeowLogger := logger.GetWhatsAppLogger("sqlstore")

	container := sqlstore.NewWithDB(db.DB, "postgres", whatsmeowLogger)

	if err := container.Upgrade(context.Background()); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			storeOp.Info().
				Str("status", "skipped").
				Msg("WhatsApp tables already exist, skipping upgrade")
		} else {
			storeOp.Failed("UPGRADE_ERROR").
				Err(err).
				Msg(logger.GetStandardizedMessage("database", "create_sqlstore", "failed"))
			return nil
		}
	} else {
		storeOp.Success().
			Msg(logger.GetStandardizedMessage("database", "create_sqlstore", "success"))
	}

	// Agora criar as relações após garantir que as tabelas do whatsmeow existam
	if err := db.createWhatsAppRelationships(); err != nil {
		db.logger.Warn().
			Str("operation", "create_relationships").
			Err(err).
			Msg("Failed to create WhatsApp relationships")
	}

	return container
}

func (db *DB) Migrate() error {
	migrator := NewMigrator(db.DB)
	return migrator.Run()
}

func (db *DB) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

func (db *DB) GetStats() sql.DBStats {
	return db.Stats()
}

func (db *DB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(tx)
	return err
}

func (db *DB) OptimizeForWhatsApp() error {
	optimizeOp := db.logger.ForOperation("optimize")

	optimizeOp.Starting().
		Msg(logger.GetStandardizedMessage("database", "optimize", "starting"))

	optimizations := []string{
		"SET statement_timeout = '30s'",
		"SET lock_timeout = '10s'",
		"SET idle_in_transaction_session_timeout = '60s'",
		"SET log_min_duration_statement = 1000",
	}

	for _, query := range optimizations {
		if _, err := db.Exec(query); err != nil {
			optimizeOp.Warn().
				Err(err).
				Str("query", query).
				Msg("Failed to apply optimization")
		}
	}

	optimizeOp.Success().
		Msg(logger.GetStandardizedMessage("database", "optimize", "success"))
	return nil
}

func (db *DB) CreateIndexes() error {
	indexOp := db.logger.ForOperation("create_indexes")

	indexOp.Starting().
		Msg(logger.GetStandardizedMessage("database", "create_indexes", "starting"))

	indexes := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_status_created ON sessions(status, created_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_jid_status ON sessions(jid, status) WHERE jid IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity) WHERE last_activity IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_webhook_url ON sessions(webhook_url) WHERE webhook_url IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_proxy_enabled ON sessions(proxy_enabled) WHERE proxy_enabled = true",
	}

	for _, query := range indexes {
		if _, err := db.Exec(query); err != nil {
			indexOp.Warn().
				Err(err).
				Str("query", query).
				Msg("Failed to create index")
		}
	}

	indexOp.Success().
		Msg(logger.GetStandardizedMessage("database", "create_indexes", "success"))
	return nil
}

func (db *DB) VerifySetup() error {
	verifyOp := db.logger.ForOperation("verify")

	verifyOp.Starting().
		Msg(logger.GetStandardizedMessage("database", "verify", "starting"))

	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'sessions'
		)
	`
	if err := db.QueryRow(query).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check sessions table: %w", err)
	}
	if !exists {
		return fmt.Errorf("sessions table does not exist")
	}

	var migrationCount int
	migrationQuery := `
		SELECT COUNT(*) FROM schema_migrations
	`
	if err := db.QueryRow(migrationQuery).Scan(&migrationCount); err != nil {
		verifyOp.Warn().
			Err(err).
			Msg("Could not check migration count (table may not exist yet)")
	} else {
		verifyOp.Info().
			Int("migrations_applied", migrationCount).
			Msg("Migration status")
	}

	verifyOp.Success().
		Msg(logger.GetStandardizedMessage("database", "verify", "success"))
	return nil
}

func (db *DB) createWhatsAppRelationships() error {
	relOp := db.logger.ForOperation("create_relationships")

	relOp.Starting().
		Msg(logger.GetStandardizedMessage("database", "create_relationships", "starting"))

	migrator := NewMigrator(db.DB)

	// Verificar se as migrações já foram aplicadas
	appliedVersions, err := migrator.GetAppliedVersions()
	if err != nil {
		// Se não conseguirmos verificar as versões aplicadas, vamos aplicar as relações diretamente
		relOp.Warn().
			Err(err).
			Msg("Failed to get applied versions, applying relationships directly")
		
		// Aplicar as relações diretamente
		return db.applyWhatsAppRelationshipsDirectly()
	}

	relationshipMigrations := []int{3, 4, 5}
	for _, version := range relationshipMigrations {
		if !appliedVersions[version] {
			relOp.Info().
				Int("version", version).
				Msg("Applying WhatsApp relationship migration")
		}
	}

	relOp.Success().
		Msg(logger.GetStandardizedMessage("database", "create_relationships", "success"))
	return nil
}

func (db *DB) applyWhatsAppRelationshipsDirectly() error {
	// Aplicar as constraints de chave estrangeira diretamente
	queries := []string{
		`DO $$
		BEGIN
			-- whatsmeow_device
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_device') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_device_session') THEN
					ALTER TABLE whatsmeow_device 
					ADD CONSTRAINT fk_whatsmeow_device_session 
					FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_identity_keys
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_identity_keys') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_identity_keys_session') THEN
					ALTER TABLE whatsmeow_identity_keys 
					ADD CONSTRAINT fk_whatsmeow_identity_keys_session 
					FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_pre_keys
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_pre_keys') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_pre_keys_session') THEN
					ALTER TABLE whatsmeow_pre_keys 
					ADD CONSTRAINT fk_whatsmeow_pre_keys_session 
					FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_sessions
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_sessions') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_sessions_session') THEN
					ALTER TABLE whatsmeow_sessions 
					ADD CONSTRAINT fk_whatsmeow_sessions_session 
					FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_sender_keys
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_sender_keys') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_sender_keys_session') THEN
					ALTER TABLE whatsmeow_sender_keys 
					ADD CONSTRAINT fk_whatsmeow_sender_keys_session 
					FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_app_state_sync_keys
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_sync_keys') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_app_state_sync_keys_session') THEN
					ALTER TABLE whatsmeow_app_state_sync_keys 
					ADD CONSTRAINT fk_whatsmeow_app_state_sync_keys_session 
					FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_app_state_version
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_version') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_app_state_version_session') THEN
					ALTER TABLE whatsmeow_app_state_version 
					ADD CONSTRAINT fk_whatsmeow_app_state_version_session 
					FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_app_state_mutation_macs
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_app_state_mutation_macs') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_app_state_mutation_macs_session') THEN
					ALTER TABLE whatsmeow_app_state_mutation_macs 
					ADD CONSTRAINT fk_whatsmeow_app_state_mutation_macs_session 
					FOREIGN KEY (jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_contacts
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_contacts') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_contacts_session') THEN
					ALTER TABLE whatsmeow_contacts 
					ADD CONSTRAINT fk_whatsmeow_contacts_session 
					FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			-- whatsmeow_chat_settings
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'whatsmeow_chat_settings') THEN
				IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints 
							  WHERE constraint_name = 'fk_whatsmeow_chat_settings_session') THEN
					ALTER TABLE whatsmeow_chat_settings 
					ADD CONSTRAINT fk_whatsmeow_chat_settings_session 
					FOREIGN KEY (our_jid) REFERENCES sessions(jid) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED;
				END IF;
			END IF;

			RAISE NOTICE 'WhatsApp relationships created successfully';
		END;
		$$ LANGUAGE plpgsql;`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to apply WhatsApp relationships: %w", err)
		}
	}

	return nil
}
