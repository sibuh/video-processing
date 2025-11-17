package initiator

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/casbin/casbin/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxadapter "github.com/pckhoi/casbin-pgx-adapter/v3"
)

type Enforcer struct {
	*casbin.Enforcer
	logger *slog.Logger
}

// NewEnforcer creates a new Casbin enforcer with PostgreSQL adapter
func NewEnforcer(pool *pgxpool.Pool, log *slog.Logger, pth string) (*Enforcer, error) {
	// Create adapter
	adapter, err := pgxadapter.NewAdapter(nil, pgxadapter.WithConnectionPool(pool))
	if err != nil {
		log.Error("failed to initialize Casbin adapter", "error", err, "path", pth)
		return nil, err
	}

	// Create enforcer
	enforcer, err := casbin.NewEnforcer(filepath.Join(pth, "model.conf"), adapter)
	if err != nil {
		log.Error("failed to initialize Casbin enforcer", "error", err, "path", pth)
		return nil, err
	}

	// Enable auto-save
	enforcer.EnableAutoSave(true)

	rules, err := readRulesFromCSV(filepath.Join(pth, "policy.csv"))
	if err != nil {
		return nil, err
	}
	for _, r := range rules {
		_, err = enforcer.AddPolicy(r[1:])
		if err != nil {
			return nil, err
		}
	}

	// Load policy after adding initial rules
	if err := enforcer.LoadPolicy(); err != nil {
		log.Error("failed to load policy", "error", err, "path", pth)
		return nil, err
	}

	return &Enforcer{
		Enforcer: enforcer,
		logger:   log,
	}, nil
}

func readRulesFromCSV(path string) ([][]string, error) {
	cleanPath := filepath.Clean(path)
	f, err := os.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read input file, error:%w", err)
	}

	defer f.Close() //nolint: errcheck

	csvReader := csv.NewReader(f)
	rules, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to read input file, error:%w", err)
	}

	return rules, nil
}
