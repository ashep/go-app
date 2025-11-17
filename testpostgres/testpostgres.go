package testpostgres

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type Option func(*Config)

func WithHost(host string) Option {
	return func(p *Config) {
		p.host = host
	}
}

func WithPort(port int) Option {
	return func(p *Config) {
		p.port = port
	}
}

func WithUser(user string) Option {
	return func(p *Config) {
		p.user = user
	}
}

func WithPassword(password string) Option {
	return func(p *Config) {
		p.password = password
	}
}

type Config struct {
	t        *testing.T
	host     string
	port     int
	user     string
	password string
}

type Postgres struct {
	dsn  string
	pool *pgxpool.Pool
}

func New(t *testing.T, opts ...Option) *Postgres {
	cfg := &Config{
		t:        t,
		host:     "postgres",
		port:     5432,
		user:     "postgres",
		password: "postgres",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	dsn := "postgres://" + cfg.user + ":" + cfg.password + "@" + cfg.host + ":" + strconv.Itoa(cfg.port)
	db, err := pgxpool.New(t.Context(), dsn+"?sslmode=disable")
	require.NoError(t, err, "failed to connect")

	dbName := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = db.Exec(t.Context(), "CREATE DATABASE "+dbName)
	require.NoError(t, err, "failed to create database "+dbName)

	testDSN := dsn + "/" + dbName + "?sslmode=disable"
	testDB, err := pgxpool.New(t.Context(), testDSN)
	require.NoError(t, err, "failed to connect to "+dbName)

	t.Cleanup(func() {
		testDB.Close()
		_, err = db.Exec(context.Background(), "DROP DATABASE "+dbName)
		require.NoError(t, err, "failed to drop database "+dbName)
		db.Close()
	})

	return &Postgres{
		dsn:  testDSN,
		pool: testDB,
	}
}

func (p *Postgres) DSN() string {
	return p.dsn
}

func (p *Postgres) DB() *pgxpool.Pool {
	return p.pool
}
