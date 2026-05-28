package governance

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigVersion holds version tracking info for a governance config file.
type ConfigVersion struct {
	ConfigName  string    `json:"config_name"`
	Version     string    `json:"version"`
	ContentHash string    `json:"content_hash"`
	LoadedAt    time.Time `json:"loaded_at"`
	Active      bool      `json:"active"`
}

// ConfigVersionService provides config file version tracking.
// Tracks SHA-256 content hashes in ops.config_versions for governance YAML configs.
type ConfigVersionService struct {
	pool *pgxpool.Pool
}

// NewConfigVersionService creates a new ConfigVersionService.
func NewConfigVersionService(pool *pgxpool.Pool) *ConfigVersionService {
	return &ConfigVersionService{pool: pool}
}

// LoadConfig reads a YAML config file, computes its SHA-256 hash, and upserts
// a version record in ops.config_versions. If the file content has changed,
// the record is updated with a new version, loaded_at timestamp, and active=true.
// Returns the current ConfigVersion or an error.
func (s *ConfigVersionService) LoadConfig(ctx context.Context, pool *pgxpool.Pool, name, filepath string) (*ConfigVersion, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", filepath, err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	version := time.Now().Format("20060102150405")

	_, err = pool.Exec(ctx, `
		INSERT INTO ops.config_versions (config_name, version, content_hash, loaded_at, active)
		VALUES ($1, $2, $3, NOW(), true)
		ON CONFLICT (config_name) DO UPDATE
		SET version = EXCLUDED.version,
		    content_hash = EXCLUDED.content_hash,
		    loaded_at = NOW(),
		    active = true
	`, name, version, hash)
	if err != nil {
		return nil, fmt.Errorf("upsert config version for %s: %w", name, err)
	}

	return &ConfigVersion{
		ConfigName:  name,
		Version:     version,
		ContentHash: hash,
		LoadedAt:    time.Now(),
		Active:      true,
	}, nil
}

// GetActiveConfig returns the current active version info for the given config name.
// Returns nil if no active version is found.
func (s *ConfigVersionService) GetActiveConfig(ctx context.Context, pool *pgxpool.Pool, name string) (*ConfigVersion, error) {
	var cv ConfigVersion
	err := pool.QueryRow(ctx, `
		SELECT config_name, version, content_hash, loaded_at, active
		FROM ops.config_versions
		WHERE config_name = $1 AND active = true
	`, name).Scan(&cv.ConfigName, &cv.Version, &cv.ContentHash, &cv.LoadedAt, &cv.Active)
	if err != nil {
		return nil, fmt.Errorf("get active config %s: %w", name, err)
	}
	return &cv, nil
}
