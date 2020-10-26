package db

import (
	"context"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/db/dbtesting"
)

// TestSurveyResponses_Create_Count tests creation and counting of db survey responses
func TestSurveyResponses_Create_Count(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	dbtesting.SetupGlobalTestDB(t)
	ctx := context.Background()

	count, err := SurveyResponses.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("Expected Count to be 0.")
	}

	_, err = SurveyResponses.Create(ctx, nil, nil, 10, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	user, err := Users.Create(ctx, NewUser{
		Email:                 "a@a.com",
		Username:              "u",
		Password:              "p",
		EmailVerificationCode: "c",
	})
	if err != nil {
		t.Fatal(err)
	}

	fakeResponse, fakeEmail := "lorem ipsum", "email@email.email"
	_, err = SurveyResponses.Create(ctx, &user.ID, nil, 9, &fakeResponse, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = SurveyResponses.Create(ctx, &user.ID, &fakeEmail, 8, nil, &fakeResponse)
	if err != nil {
		t.Fatal(err)
	}

	_, err = SurveyResponses.Create(ctx, nil, &fakeEmail, 8, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	count, err = SurveyResponses.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatal("Expected Count to be 4.")
	}
}
