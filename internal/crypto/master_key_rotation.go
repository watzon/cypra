package crypto

//revive:disable:exported

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
)

var ErrRotationIncomplete = errors.New("master key rotation incomplete")

type EncryptedColumn struct {
	Table     string
	IDColumn  string
	DEKColumn string
}

type RotationOptions struct {
	BatchSize int
	MaxRows   int64
}

type RotationService struct {
	db      *sql.DB
	targets []EncryptedColumn
}

func NewRotationService(db *sql.DB, targets []EncryptedColumn) (*RotationService, error) {
	for _, target := range targets {
		if err := validateTarget(target); err != nil {
			return nil, err
		}
	}
	return &RotationService{db: db, targets: targets}, nil
}

func (s *RotationService) Start(ctx context.Context, oldKEK, newKEK []byte, opts RotationOptions) (uuid.UUID, error) {
	rotationID := uuid.New()
	rowsTotal, err := s.countRows(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO master_key_rotations (id, phase, rows_total, rows_done) VALUES ($1, 'rewrap', $2, 0)`, rotationID, rowsTotal); err != nil {
		return uuid.Nil, fmt.Errorf("insert master key rotation: %w", err)
	}
	return rotationID, s.Resume(ctx, rotationID, oldKEK, newKEK, opts)
}

func (s *RotationService) Resume(ctx context.Context, rotationID uuid.UUID, oldKEK, newKEK []byte, opts RotationOptions) error {
	if len(oldKEK) != MasterKeyBytes || len(newKEK) != MasterKeyBytes {
		return ErrInvalidMasterKey
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}
	rowsDone, rowsTotal, phase, err := s.loadState(ctx, rotationID)
	if err != nil {
		return err
	}
	if phase == "done" {
		return nil
	}
	processedThisRun := int64(0)
	seen := int64(0)
	for _, target := range s.targets {
		rows, err := s.targetRows(ctx, target)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if seen >= rowsTotal {
				break
			}
			if seen < rowsDone {
				seen++
				continue
			}
			if opts.MaxRows > 0 && processedThisRun >= opts.MaxRows {
				return ErrRotationIncomplete
			}
			if err := s.rewrapRow(ctx, target, row, oldKEK, newKEK); err != nil {
				return err
			}
			seen++
			processedThisRun++
			if _, err := s.db.ExecContext(ctx, `UPDATE master_key_rotations SET rows_done = $1 WHERE id = $2`, seen, rotationID); err != nil {
				return fmt.Errorf("update rotation progress: %w", err)
			}
		}
	}
	if seen < rowsTotal {
		return fmt.Errorf("rotation row count drift: saw %d rows, expected at least %d", seen, rowsTotal)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE master_key_rotations SET phase = 'cutover' WHERE id = $1`, rotationID)
	if err != nil {
		return fmt.Errorf("mark rotation cutover: %w", err)
	}
	return nil
}

func (s *RotationService) ConfirmCutover(ctx context.Context, rotationID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `UPDATE master_key_rotations SET phase = 'done', completed_at = now(), error = NULL WHERE id = $1`, rotationID)
	if err != nil {
		return fmt.Errorf("confirm rotation cutover: %w", err)
	}
	return nil
}

type encryptedRow struct {
	id           uuid.UUID
	encryptedDEK []byte
}

func (s *RotationService) countRows(ctx context.Context) (int64, error) {
	var total int64
	for _, target := range s.targets {
		query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s IS NOT NULL", target.Table, target.DEKColumn) // #nosec G201 -- identifiers are validated by validateTarget.
		var count int64
		if err := s.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return 0, fmt.Errorf("count encrypted rows in %s: %w", target.Table, err)
		}
		total += count
	}
	return total, nil
}

func (s *RotationService) loadState(ctx context.Context, rotationID uuid.UUID) (int64, int64, string, error) {
	var rowsDone int64
	var rowsTotal int64
	var phase string
	err := s.db.QueryRowContext(ctx, `SELECT rows_done, rows_total, phase FROM master_key_rotations WHERE id = $1`, rotationID).Scan(&rowsDone, &rowsTotal, &phase)
	if err != nil {
		return 0, 0, "", fmt.Errorf("load rotation state: %w", err)
	}
	return rowsDone, rowsTotal, phase, nil
}

func (s *RotationService) targetRows(ctx context.Context, target EncryptedColumn) ([]encryptedRow, error) {
	query := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s IS NOT NULL ORDER BY %s", target.IDColumn, target.DEKColumn, target.Table, target.DEKColumn, target.IDColumn) // #nosec G201 -- identifiers are validated by validateTarget.
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("select encrypted rows from %s: %w", target.Table, err)
	}
	defer func() { _ = rows.Close() }()
	var result []encryptedRow
	for rows.Next() {
		var row encryptedRow
		if err := rows.Scan(&row.id, &row.encryptedDEK); err != nil {
			return nil, fmt.Errorf("scan encrypted row: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate encrypted rows: %w", err)
	}
	return result, nil
}

func (s *RotationService) rewrapRow(ctx context.Context, target EncryptedColumn, row encryptedRow, oldKEK, newKEK []byte) error {
	rewrapped, err := RewrapDEK(row.encryptedDEK, oldKEK, newKEK)
	if err != nil {
		return err
	}
	query := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", target.Table, target.DEKColumn, target.IDColumn) // #nosec G201 -- identifiers are validated by validateTarget.
	if _, err := s.db.ExecContext(ctx, query, rewrapped, row.id); err != nil {
		return fmt.Errorf("update encrypted dek in %s: %w", target.Table, err)
	}
	return nil
}

var safeIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func validateTarget(target EncryptedColumn) error {
	if !safeIdentifier.MatchString(target.Table) || !safeIdentifier.MatchString(target.IDColumn) || !safeIdentifier.MatchString(target.DEKColumn) {
		return fmt.Errorf("unsafe encrypted-column target: %+v", target)
	}
	return nil
}
