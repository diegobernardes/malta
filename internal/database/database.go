package database

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"
)

// Transaction is used to generate SQL transactions.
type Transaction interface {
	Begin(ctx context.Context, readOnly bool, level sql.IsolationLevel) (*sql.Tx, error)
}

// TransactionHandler is used to commit or rollback the transaction in case of errors.
func TransactionHandler(logger zerolog.Logger) func(*sql.Tx, error) error {
	return func(tx *sql.Tx, err error) error {
		if p := recover(); p != nil {
			if txerr := tx.Rollback(); txerr != nil {
				logger.Error().Err(txerr).Msg("failed to rollback the transaction")
			}
			logger.Error().Interface("recoverValue", p).Msg("panic detected")
			panic(p)
		}

		if err != nil {
			logger.Error().Err(err).Msg("error detect, rolling back the transaction")
			if txerr := tx.Rollback(); txerr != nil {
				logger.Error().Err(txerr).Msg("failed to rollback the transaction")
			}
			return err
		}

		if err := tx.Commit(); err != nil {
			logger.Error().Err(err).Msg("failed during transaction commit")
			return err
		}
		return nil
	}
}
