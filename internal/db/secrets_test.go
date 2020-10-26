package db

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

func TestAllByKeyValue(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	key := "arthur dent"
	value := "heart of gold"
	err := Secrets.InsertKeyValue(ctx, key, value)
	if err != nil {
		t.Fatal(err)
	}

	sec, err := Secrets.GetByKeyName(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if sec.KeyName.String != key {
		t.Fatalf("Expected %s received %s", value, sec.Value)
	}

	newVal := "infinite improbability drive"
	err = Secrets.UpdateByKeyName(ctx, key, newVal)
	if err != nil {
		t.Fatal(err)
	}

	s, err := Secrets.GetByKeyName(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if s.Value != newVal {
		t.Fatalf("Expected %s received %s", newVal, s.Value)
	}

	err = Secrets.DeleteByKeyName(ctx, key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Secrets.GetByKeyName(ctx, key)
	if err == nil {
		t.Fatal("Secret not deleted from database.")
	}
}

func TestAllByKeySource(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	sourceType := "bitbucket"
	var sourceID int32 = 42
	value := "Life, the Univese, and Everything"
	err := Secrets.InsertSourceTypeValue(ctx, sourceType, int32(sourceID), value)
	if err != nil {
		t.Fatal(err)
	}

	sec, err := Secrets.GetBySource(ctx, sourceType, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if sec.Value != value {
		t.Fatalf("Expected %s received %s", value, sec.Value)
	}

	newVal := "infinite improbability drive"
	err = Secrets.UpdateBySource(ctx, sourceType, sourceID, newVal)
	if err != nil {
		t.Fatal(err)
	}

	s, err := Secrets.GetBySource(ctx, sourceType, sourceID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Value != newVal {
		t.Fatalf("Expected %s received %s", newVal, s.Value)
	}

	err = Secrets.DeleteBySource(ctx, sourceType, sourceID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Secrets.GetBySource(ctx, sourceType, sourceID)
	if err == nil {
		t.Fatal("Secret not deleted from database.")
	}
}

func TestAllByID(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	key := "arthur dent"
	value := "heart of gold"
	err := Secrets.InsertKeyValue(ctx, key, value)
	if err != nil {
		t.Fatal(err)
	}

	s, _ := Secrets.GetByKeyName(ctx, key)
	sec, err := Secrets.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatal(err)
	}

	newVal := "infinite improbability drive"
	err = Secrets.UpdateByID(ctx, sec.ID, newVal)
	if err != nil {
		t.Fatal(err)
	}

	s, err = Secrets.GetByID(ctx, sec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Value != newVal {
		t.Fatalf("Expected %s received %s", newVal, s.Value)
	}

	err = Secrets.DeleteByID(ctx, sec.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Secrets.GetByID(ctx, sec.ID)
	if err == nil { // to fail this would error since we're removing the object
		t.Fatal("Secret not deleted from database.")
	}
}
