package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/felipe/zemeow/internal/config"
	"github.com/felipe/zemeow/internal/db/migrations"
	"github.com/felipe/zemeow/internal/logger"
	_ "github.com/lib/pq"
	"go.mau.fi/whatsmeow/store/sqlstore"
)

// DB representa a conexão com o banco de dados
type DB struct {
	*sql.DB
	config *config.DatabaseConfig
	logger logger.Logger
}

// Connect cria uma nova conexão com o banco usando a configuração completa
func Connect(cfg *config.Config) (*DB, error) {
	return New(&cfg.Database)
}

// Migrate executa as migrações do banco de dados (alias para compatibilidade)
func Migrate(db *DB) error {
	return db.Migrate()
}

// New cria uma nova conexão com o banco de dados
func New(cfg *config.DatabaseConfig) (*DB, error) {
	log := logger.Get()

	log.Info().Str("host", cfg.Host).Int("port", cfg.Port).Str("database", cfg.Name).Msg("Connecting to database")

	// Abrir conexão com PostgreSQL
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configurar pool de conexões
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Testar conexão
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().Msg("Database connection established successfully")

	return &DB{
		DB:     db,
		config: cfg,
		logger: log,
	}, nil
}

// Close fecha a conexão com o banco de dados
func (db *DB) Close() error {
	db.logger.Info().Msg("Closing database connection")
	return db.DB.Close()
}

// GetSQLStore retorna um store compatível com whatsmeow
func (db *DB) GetSQLStore() *sqlstore.Container {
	// Criar logger específico para whatsmeow
	whatsmeowLogger := logger.GetWhatsAppLogger("sqlstore")

	// Criar container do sqlstore
	container := sqlstore.NewWithDB(db.DB, "postgres", whatsmeowLogger)

	// Executar upgrade das tabelas do whatsmeow
	if err := container.Upgrade(); err != nil {
		db.logger.Error().Err(err).Msg("Failed to upgrade whatsmeow store")
		return nil
	}

	db.logger.Info().Msg("WhatsApp SQL store container created and upgraded")
	return container
}

// Migrate executa as migrações do banco de dados
func (db *DB) Migrate() error {
	migrator := migrations.NewMigrator(db.DB)
	return migrator.Run()
}

// Health verifica a saúde da conexão com o banco
func (db *DB) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

// GetStats retorna estatísticas da conexão
func (db *DB) GetStats() sql.DBStats {
	return db.Stats()
}

// Transaction executa uma função dentro de uma transação
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

// OptimizeForWhatsApp aplica configurações otimizadas para WhatsApp
func (db *DB) OptimizeForWhatsApp() error {
	db.logger.Info().Msg("Applying WhatsApp optimizations to PostgreSQL")

	optimizations := []string{
		// Configurações de performance para WhatsApp
		"SET statement_timeout = '30s'",
		"SET lock_timeout = '10s'",
		"SET idle_in_transaction_session_timeout = '60s'",

		// Configurações de logging
		"SET log_min_duration_statement = 1000", // Log queries > 1s
		"SET log_checkpoints = on",
		"SET log_connections = on",
		"SET log_disconnections = on",

		// Configurações de autovacuum
		"SET autovacuum_vacuum_scale_factor = 0.1",
		"SET autovacuum_analyze_scale_factor = 0.05",
	}

	for _, query := range optimizations {
		if _, err := db.Exec(query); err != nil {
			db.logger.Warn().Err(err).Str("query", query).Msg("Failed to apply optimization")
			// Não retornar erro, apenas log warning
		}
	}

	db.logger.Info().Msg("WhatsApp optimizations applied")
	return nil
}

// CreateIndexes cria índices otimizados para WhatsApp
func (db *DB) CreateIndexes() error {
	db.logger.Info().Msg("Creating optimized indexes for WhatsApp")

	indexes := []string{
		// Índices para tabelas do whatsmeow
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_whatsmeow_device_jid ON whatsmeow_device(jid)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_whatsmeow_identity_address ON whatsmeow_identity_keys(our_jid, their_id)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_whatsmeow_prekeys_jid ON whatsmeow_pre_keys(jid)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_whatsmeow_sessions_jid ON whatsmeow_sessions(our_jid, their_jid)",

		// Índices para tabela sessions (já criados nas migrations, mas garantindo)
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_status_created ON sessions(status, created_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_jid_status ON sessions(jid, status) WHERE jid IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity) WHERE last_activity IS NOT NULL",
	}

	for _, query := range indexes {
		if _, err := db.Exec(query); err != nil {
			db.logger.Warn().Err(err).Str("query", query).Msg("Failed to create index")
			// Não retornar erro, apenas log warning
		}
	}

	db.logger.Info().Msg("Optimized indexes created")
	return nil
}
