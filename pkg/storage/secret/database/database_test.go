package database

import (
	"testing"

	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/storage/secret/migrator"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestBegin(t *testing.T) {
	t.Run("rollback transaction", func(t *testing.T) {
		db := setup(t)
		_, err := db.ExecContext(t.Context(), "CREATE TABLE storage_secret_database_test (x int)")
		require.NoError(t, err)
		txCtx, tx, err := db.Begin(t.Context())
		require.NoError(t, err)
		// Insert a new row
		_, err = db.ExecContext(txCtx, "INSERT INTO storage_secret_database_test VALUES (1)")
		require.NoError(t, err)
		require.NoError(t, tx.Rollback())
		// Query the table after rollback
		rows, err := db.QueryContext(t.Context(), "SELECT COUNT(*) FROM storage_secret_database_test")
		require.NoError(t, err)
		// Insertion was rolledback
		require.True(t, rows.Next())
		var x int
		require.NoError(t, rows.Scan(&x))
		require.Equal(t, 0, x)
	})

	t.Run("commit transaction", func(t *testing.T) {
		db := setup(t)
		_, err := db.ExecContext(t.Context(), "CREATE TABLE storage_secret_database_test (x int)")
		require.NoError(t, err)
		txCtx, tx, err := db.Begin(t.Context())
		require.NoError(t, err)
		// Insert a new row
		_, err = db.ExecContext(txCtx, "INSERT INTO storage_secret_database_test VALUES (1)")
		require.NoError(t, err)
		require.NoError(t, tx.Commit())
		// Query the table after commit
		rows, err := db.QueryContext(t.Context(), "SELECT COUNT(*) FROM storage_secret_database_test")
		require.NoError(t, err)
		// Insertion was committed
		require.True(t, rows.Next())
		var x int
		require.NoError(t, rows.Scan(&x))
		require.Equal(t, 1, x)
	})

	t.Run("starting a transaction inside another transaction just reuses the first transaction", func(t *testing.T) {
		db := setup(t)
		_, err := db.ExecContext(t.Context(), "CREATE TABLE storage_secret_database_test (x int)")
		require.NoError(t, err)
		ctx1, tx1, err := db.Begin(t.Context())
		require.NoError(t, err)
		ctx2, tx2, err := db.Begin(ctx1)
		require.NoError(t, err)
		require.Equal(t, tx1.(*Tx).tx, tx2.(*Tx).tx)
		_, err = db.ExecContext(ctx2, "INSERT INTO storage_secret_database_test VALUES (1)")
		require.NoError(t, err)
		// No-op since it is a nested tx
		require.NoError(t, tx2.Commit())
		// Query to check if tx has been committed
		rows, err := db.QueryContext(t.Context(), "SELECT COUNT(*) FROM storage_secret_database_test")
		require.NoError(t, err)
		require.True(t, rows.Next())
		var x int
		require.NoError(t, rows.Scan(&x))
		require.Equal(t, 0, x)
		// Actually commits because it is the first tx
		require.NoError(t, tx1.Commit())
		// Query to check if tx has been committed
		rows, err = db.QueryContext(t.Context(), "SELECT COUNT(*) FROM storage_secret_database_test")
		require.NoError(t, err)
		require.True(t, rows.Next())
		require.NoError(t, rows.Scan(&x))
		require.Equal(t, 1, x)
	})
}

func setup(t *testing.T) *Database {
	tracer := noop.NewTracerProvider().Tracer("test")
	testDB := sqlstore.NewTestStore(t, sqlstore.WithMigrator(migrator.New()))
	return ProvideDatabase(testDB, tracer)
}
