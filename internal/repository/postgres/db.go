package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func New(dsn string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	dbpool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	// Ping the DB to ensure connection is valid
	if err := dbpool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	log.Println("Connected to PostgreSQL successfully")

	// In a real production app you'd run migrations (golang-migrate, goose, etc.)
	// Here we just auto-create tables for the template
	err = autoMigrate(dbpool)
	if err != nil {
		return nil, fmt.Errorf("auto migration failed: %w", err)
	}

	return dbpool, nil
}

func autoMigrate(db *pgxpool.Pool) error {
	ctx := context.Background()

	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(50) DEFAULT 'user',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notes (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			content TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	return seedAdmin(ctx, db)
}

func seedAdmin(ctx context.Context, db *pgxpool.Pool) error {
	var count int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("golang123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		_, err = db.Exec(ctx, "INSERT INTO users (username, password, role) VALUES ($1, $2, $3)", "admin", string(hashedPassword), "admin")
		if err != nil {
			return err
		}
		log.Println("🌱 Seeded default admin user  [ admin : golang123 ]")
	}

	return nil
}
