// Package database provides PostgreSQL connection management and pooling for the SwaRupa API.
// It initializes a pgx connection pool from environment configuration and maintains
// database lifecycle hooks (Connect/Close) for server startup and graceful shutdown.
package database

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the global PostgreSQL connection pool shared across all request handlers.
// This is a thread-safe pool maintained by pgx that manages multiple concurrent database connections.
// All database queries in handlers execute through this pool, which automatically handles
// connection lifecycle, retry logic, and resource cleanup.
var DB *pgxpool.Pool

// Connect initializes the PostgreSQL connection pool by parsing the POOLER_DATABASE_URL
// environment variable and creating a pgxpool configuration with optimized settings.
//
// Configuration:
// - MaxConns: 10 (maximum concurrent connections allowed in the pool)
// - MinConns: 2 (minimum warm connections maintained)
// - MaxConnLifetime: 1 hour (connections are recycled after this duration)
// - MaxConnIdleTime: 30 minutes (idle connections are closed after this duration)
//
// Connection String Format:
// The POOLER_DATABASE_URL environment variable should contain a PostgreSQL connection URI:
// postgres://username:password@host:port/database?sslmode=require
//
// SQL Connection Details:
// The connection pool communicates with PostgreSQL using the native pgx driver over TCP.
// Each physical connection in the pool can execute parameterized SQL queries and transactions.
// The pool automatically reuses connections for subsequent requests, improving performance.
//
// Error Handling:
// This function calls log.Fatal() if:
// - POOLER_DATABASE_URL environment variable is missing or empty
// - Connection string parsing fails
// - Initial connection to the database server fails after 5 seconds
//
// The function performs a ping verification to ensure the database is reachable before returning.
func Connect() {
	// Fetch the PostgreSQL connection URL from the POOLER_DATABASE_URL environment variable.
	// This is set during deployment or in a .env file for local development.
	// Using environment variables for secrets is a security best practice (12-factor app methodology).
	dsn := os.Getenv("POOLER_DATABASE_URL")
	if dsn == "" {
		// If the connection string is not set, fail fast rather than attempting a default connection.
		// This ensures the application cannot run without proper database configuration.
		log.Fatal("DATABASE_URL is not set")
	}

	// Parse the PostgreSQL connection string into a pgxpool.Config object.
	// pgxpool.ParseConfig handles URL parsing, credential extraction, and TLS configuration.
	// The config can then be customized before creating the actual connection pool.
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// Connection string parsing errors indicate invalid URL format or missing required components.
		// Fail immediately rather than attempting to connect with bad configuration.
		log.Fatalf("Unable to parse DATABASE_URL: %v\n", err)
	}

	// Configure connection pool parameters for optimal resource utilization.
	// Pool tuning depends on application concurrency and database server limits.
	config.MaxConns = 10                      // Maximum concurrent connections allowed
	config.MinConns = 2                       // Maintain at least 2 warm connections for quick query response
	config.MaxConnLifetime = time.Hour        // Recycle connections after 1 hour to refresh authentication
	config.MaxConnIdleTime = 30 * time.Minute // Close idle connections after 30 minutes

	// Create the actual connection pool from the configured settings.
	// NewWithConfig establishes initial connections (MinConns) and prepares to scale up to MaxConns.
	// The pool is thread-safe and designed for concurrent usage across multiple goroutines.
	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		// Pool creation errors indicate resource exhaustion or system-level connectivity issues.
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	// Verify the database connection is actually working before returning.
	// This catches configuration errors early rather than at the first query execution.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() // Always clean up context resources to prevent leaks

	// Ping executes a trivial query (SELECT 1) to verify connectivity and authentication.
	// A timeout of 5 seconds prevents the function from blocking indefinitely.
	if err := DB.Ping(ctx); err != nil {
		// If the database is unreachable or unresponsive, fail fast and shutdown the server.
		log.Fatalf("Failed to connect to database: %v\n", err)
	}

	// Log successful connection for operational visibility and debugging.
	log.Println("Connected to Supabase PostgreSQL")
}

// Close gracefully shuts down the PostgreSQL connection pool.
// This function should be deferred immediately after Connect() completes successfully.
// It waits for all active connections and queries to complete before closing the pool,
// allowing for clean server shutdown without abandoning in-flight operations.
//
// Behavior:
// - Prevents new connections from being acquired
// - Waits for all currently-checked-out connections to be returned
// - Closes all idle connections
// - Logs completion status
//
// This function is safe to call multiple times (idempotent).
func Close() {
	if DB != nil {
		// Close the global connection pool.
		// This signals all idle connections to shut down gracefully.
		// In-flight queries are allowed to complete before closure.
		DB.Close()
		// Log the closure for operational visibility and debugging.
		log.Println("Database connection closed")
	}
}
