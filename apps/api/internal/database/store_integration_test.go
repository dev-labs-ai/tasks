//go:build integration

package database_test

import (
	"errors"
	"os"
	"testing"

	"github.com/dev-labs-ai/tasks/apps/api/internal/database"
	"github.com/dev-labs-ai/tasks/apps/api/internal/repositories"
	"github.com/dev-labs-ai/tasks/apps/api/internal/testdb"
)

func TestMain(m *testing.M) {
	os.Exit(testdb.Run(m))
}

func TestInTxCommitsOrRollsBack(t *testing.T) {
	ctx := t.Context()
	store := database.NewStore(testdb.NewMigratedDatabase(t))

	errBoom := errors.New("boom")
	err := store.InTx(ctx, func(q repositories.Querier) error {
		if _, err := q.CreateUser(ctx, repositories.CreateUserParams{Email: "rolled@example.com", PasswordHash: "h"}); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("InTx err = %v, want the callback's error", err)
	}
	if _, err := store.GetUserByEmail(ctx, "rolled@example.com"); err == nil {
		t.Error("the user survived a rolled-back transaction")
	}

	err = store.InTx(ctx, func(q repositories.Querier) error {
		_, err := q.CreateUser(ctx, repositories.CreateUserParams{Email: "kept@example.com", PasswordHash: "h"})
		return err
	})
	if err != nil {
		t.Fatalf("InTx: %v", err)
	}
	if _, err := store.GetUserByEmail(ctx, "kept@example.com"); err != nil {
		t.Errorf("the committed user is missing: %v", err)
	}
}
