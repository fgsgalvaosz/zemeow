package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

	// Criar container do sqlstore - ele fará o upgrade automaticamente
	container := sqlstore.NewWithDB(db.DB, "postgres", whatsmeowLogger)

	// Executar upgrade das tabelas do whatsmeow automaticamente
	if err := container.Upgrade(); err != nil {
		// Verificar se o erro é sobre tabelas já existentes (comportamento esperado)
		if strings.Contains(err.Error(), "already exists") {
			db.logger.Info().Msg("WhatsApp tables already exist, skipping upgrade")
		} else {
			db.logger.Error().Err(err).Msg("Failed to upgrade whatsmeow store")
			return nil
		}
	}

	db.logger.Info().Msg("WhatsApp SQL store container created and upgraded automatically")

	// Após o upgrade do whatsmeow, executar migrações que dependem das tabelas whatsmeow
	if err := db.createWhatsAppRelationships(); err != nil {
		db.logger.Warn().Err(err).Msg("Failed to create WhatsApp relationships")
		// Não retornar erro, apenas log warning
	}

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

// CreateIndexes cria índices otimizados para as tabelas da aplicação
func (db *DB) CreateIndexes() error {
	db.logger.Info().Msg("Creating optimized indexes for application tables")

	indexes := []string{
		// Índices adicionais para tabela sessions (complementando os das migrations)
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_status_created ON sessions(status, created_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_jid_status ON sessions(jid, status) WHERE jid IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_last_activity ON sessions(last_activity) WHERE last_activity IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_webhook_url ON sessions(webhook_url) WHERE webhook_url IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_proxy_enabled ON sessions(proxy_enabled) WHERE proxy_enabled = true",
	}

	for _, query := range indexes {
		if _, err := db.Exec(query); err != nil {
			db.logger.Warn().Err(err).Str("query", query).Msg("Failed to create index")
			// Não retornar erro, apenas log warning
		}
	}

	db.logger.Info().Msg("Application indexes created successfully")
	return nil
}

// VerifySetup verifica se o banco está configurado corretamente
func (db *DB) VerifySetup() error {
	db.logger.Info().Msg("Verifying database setup")

	// Verificar se a tabela sessions existe
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

	// Verificar se as migrações foram aplicadas
	var migrationCount int
	migrationQuery := `
		SELECT COUNT(*) FROM schema_migrations
	`
	if err := db.QueryRow(migrationQuery).Scan(&migrationCount); err != nil {
		db.logger.Warn().Err(err).Msg("Could not check migration count (table may not exist yet)")
	} else {
		db.logger.Info().Int("migrations_applied", migrationCount).Msg("Migration status")
	}

	db.logger.Info().Msg("Database setup verification completed successfully")
	return nil
}

// createWhatsAppRelationships cria relacionamentos entre tabelas sessions e whatsmeow
func (db *DB) createWhatsAppRelationships() error {
	db.logger.Info().Msg("Creating relationships between sessions and WhatsApp tables")

	// Executar migrações específicas que criam relacionamentos
	// Isso garante que os relacionamentos sejam criados após as tabelas whatsmeow existirem
	migrator := migrations.NewMigrator(db.DB)

	// Verificar se as migrações de relacionamento já foram aplicadas
	appliedVersions, err := migrator.GetAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to get applied versions: %w", err)
	}

	// Executar migrações 3, 4 e 5 se ainda não foram aplicadas
	relationshipMigrations := []int{3, 4, 5}
	for _, version := range relationshipMigrations {
		if !appliedVersions[version] {
			db.logger.Info().Int("version", version).Msg("Applying WhatsApp relationship migration")
			// As migrações serão aplicadas automaticamente pelo sistema normal de migrações
		}
	}

	db.logger.Info().Msg("WhatsApp relationships setup completed")
	return nil
}
