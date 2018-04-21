package storers

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-uuid"

	"impractical.co/auth/scopes"
	"impractical.co/pqarrays"
)

const (
	changeUserPolicy = 1 << iota
	changeUserExceptions
	changeClientPolicy
	changeClientExceptions
	changeIsDefault
	changeVariations
)

type StorerFactory interface {
	NewStorer(ctx context.Context) (scopes.Storer, error)
	TeardownStorers() error
}

var storerFactories []StorerFactory

func uuidOrFail(t *testing.T) string {
	t.Helper()
	id, err := uuid.GenerateUUID()
	if err != nil {
		t.Fatalf("Unexpected error generating ID: %s", err.Error())
	}
	return id
}

func TestMain(m *testing.M) {
	flag.Parse()
	result := m.Run()
	for _, factory := range storerFactories {
		err := factory.TeardownStorers()
		if err != nil {
			log.Printf("Error cleaning up after %T: %s", factory, err.Error())
		}
	}
	os.Exit(result)
}

func runTest(t *testing.T, f func(*testing.T, scopes.Storer, context.Context)) {
	t.Parallel()
	for _, factory := range storerFactories {
		ctx := context.Background()
		storer, err := factory.NewStorer(ctx)
		if err != nil {
			t.Fatalf("Error creating Storer from %T: %s", factory, err.Error())
		}
		t.Run(fmt.Sprintf("Storer=%T", storer), func(t *testing.T) {
			t.Parallel()
			f(t, storer, ctx)
		})
	}
}

func TestCreateAndGetScope(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		scope := scopes.Scope{
			ID:         "https://scopes.impractical.co/test",
			UserPolicy: "DEFAULT_ALLOW",
			UserExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
				uuidOrFail(t),
			},
			ClientPolicy: "DEFAULT_ALLOW",
			ClientExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
			},
			IsDefault: true,
		}
		err := storer.Create(ctx, scope)
		if err != nil {
			t.Fatalf("Unexpected error creating scope: %s", err.Error())
		}

		resps, err := storer.GetMulti(ctx, []string{scope.ID})
		if err != nil {
			t.Fatalf("Unexpected error retrieving scope: %s", err.Error())
		}
		resp, ok := resps[scope.ID]
		if !ok {
			t.Fatalf("Scope not found.")
		}
		if diff := cmp.Diff(scope, resp); diff != "" {
			t.Errorf("Retrieved scope doesn't match expectation:\n%s", diff)
		}
	})
}

func TestGetNonexistentScope(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		res, err := storer.GetMulti(ctx, []string{"nope"})
		if err != nil {
			t.Fatalf("Unexpected error: %s", err.Error())
		}
		if len(res) != 0 {
			t.Fatalf("Expected 0 results, got %+v", res)
		}
	})
}

func TestCreateDuplicateID(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		scope := scopes.Scope{
			ID:         "https://scopes.impractical.co/test",
			UserPolicy: "DEFAULT_ALLOW",
			UserExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
				uuidOrFail(t),
			},
			ClientPolicy: "DEFAULT_ALLOW",
			ClientExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
			},
			IsDefault: true,
		}
		err := storer.Create(ctx, scope)
		if err != nil {
			t.Fatalf("Unexpected error creating scope: %s", err.Error())
		}

		scope2 := scopes.Scope{
			ID:         "https://scopes.impractical.co/test",
			UserPolicy: "DEFAULT_DENY",
			UserExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
				uuidOrFail(t),
			},
			ClientPolicy: "DEFAULT_DENY",
			ClientExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
			},
			IsDefault: false,
		}

		err = storer.Create(ctx, scope2)
		if err != scopes.ErrScopeAlreadyExists {
			t.Fatalf("Expected ErrScopeAlreadyExists, got %s", err.Error())
		}

		// we shouldn't have changed anything about what was stored
		resps, err := storer.GetMulti(ctx, []string{scope.ID})
		if err != nil {
			t.Fatalf("Unexpected error retrieving scope: %s", err.Error())
		}
		resp, ok := resps[scope.ID]
		if !ok {
			t.Fatalf("Scope not found.")
		}
		if diff := cmp.Diff(scope, resp); diff != "" {
			t.Errorf("Retrieved scope doesn't match expectation:\n%s", diff)
		}

	})
}

func TestCreateMultipleScopes(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		scope := scopes.Scope{
			ID:         "https://scopes.impractical.co/test",
			UserPolicy: "DEFAULT_DENY",
			UserExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
				uuidOrFail(t),
			},
			ClientPolicy: "DEFAULT_DENY",
			ClientExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
			},
			IsDefault: false,
		}
		err := storer.Create(ctx, scope)
		if err != nil {
			t.Fatalf("Unexpected error creating scope: %s", err.Error())
		}
		scope2 := scope
		scope2.ID = "https://scopes.impractical.co/test2"
		err = storer.Create(ctx, scope2)
		if err != nil {
			t.Fatalf("Unexpected error creating scope: %s", err.Error())
		}

		resps, err := storer.GetMulti(ctx, []string{scope.ID, scope2.ID})
		if err != nil {
			t.Fatalf("Unexpected error retrieving scopes: %s", err.Error())
		}
		resp1, ok := resps[scope.ID]
		if !ok {
			t.Fatalf("Scope %q not found.", scope.ID)
		}
		resp2, ok := resps[scope2.ID]
		if !ok {
			t.Fatalf("Scope %q not found.", scope2.ID)
		}
		if diff := cmp.Diff(scope, resp1); diff != "" {
			t.Errorf("Retrieved scope %q doesn't match expectation:\n%s", scope.ID, diff)
		}
		if diff := cmp.Diff(scope2, resp2); diff != "" {
			t.Errorf("Retrieved scope %q doesn't match expectation:\n%s", scope2.ID, diff)
		}
	})
}

func TestListDefault(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		testScopes := []scopes.Scope{
			{
				ID:         "https://scopes.impractical.co/test",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: false,
			},
			{
				ID:         "https://scopes.impractical.co/test2",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: true,
			},
			{
				ID:         "https://scopes.impractical.co/test3",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: false,
			},
			{
				ID:         "https://scopes.impractical.co/test4",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: true,
			},
		}
		for _, scope := range testScopes {
			err := storer.Create(ctx, scope)
			if err != nil {
				t.Fatalf("Unexpected error creating scope %q: %s", scope.ID, err.Error())
			}
		}

		results, err := storer.ListDefault(ctx)
		if err != nil {
			t.Fatalf("Unexpected error listing scopes: %s", err.Error())
		}

		expectations := []scopes.Scope{testScopes[1], testScopes[3]}
		scopes.ByID(expectations)

		if diff := cmp.Diff(expectations, results); diff != "" {
			t.Errorf("Unexpected results for listing scopes:\n%s", diff)
		}
	})
}

func TestUpdateOneOfMany(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		throwaways := []scopes.Scope{
			{
				ID:         "https://scopes.impractical.co/test",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: false,
			},
			{
				ID:         "https://scopes.impractical.co/test2",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: true,
			},
			{
				ID:         "https://scopes.impractical.co/test3",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: false,
			},
			{
				ID:         "https://scopes.impractical.co/test4",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: true,
			},
		}
		var ids []string
		for _, scope := range throwaways {
			err := storer.Create(ctx, scope)
			if err != nil {
				t.Fatalf("Unexpected error creating scope %q: %s", scope.ID, err.Error())
			}
			ids = append(ids, scope.ID)
		}

		for i := 1; i < changeVariations; i++ {
			i := i
			t.Run(fmt.Sprintf("i=%d", i), func(t *testing.T) {
				t.Parallel()

				scope := scopes.Scope{
					ID:         "https://scopes.impractical.co/updated/" + strconv.Itoa(i),
					UserPolicy: "DEFAULT_DENY",
					UserExceptions: pqarrays.StringArray{
						uuidOrFail(t),
						uuidOrFail(t),
						uuidOrFail(t),
					},
					ClientPolicy: "DEFAULT_DENY",
					ClientExceptions: pqarrays.StringArray{
						uuidOrFail(t),
						uuidOrFail(t),
					},
					IsDefault: true,
				}
				err := storer.Create(ctx, scope)
				if err != nil {
					t.Fatalf("Unexpected error creating scope %q: %s", scope.ID, err.Error())
				}
				scopeIDs := append([]string{}, ids...)
				scopeIDs = append(scopeIDs, scope.ID)

				var change scopes.Change
				if i&changeUserPolicy != 0 {
					userPolicy := scopes.PolicyAllowAll
					change.UserPolicy = &userPolicy
				}
				if i&changeUserExceptions != 0 {
					userExceptions := []string{"user1", "user2"}
					change.UserExceptions = &userExceptions
				}
				if i&changeClientPolicy != 0 {
					clientPolicy := scopes.PolicyDenyAll
					change.ClientPolicy = &clientPolicy
				}
				if i&changeClientExceptions != 0 {
					clientExceptions := []string{"client1", "client2", "client3"}
					change.ClientExceptions = &clientExceptions
				}
				if i&changeIsDefault != 0 {
					isDefault := false
					change.IsDefault = &isDefault
				}
				expectation := scopes.Apply(change, scope)

				err = storer.Update(ctx, scope.ID, change)
				if err != nil {
					t.Fatalf("Unexpected error updating scope: %s", err.Error())
				}

				res, err := storer.GetMulti(ctx, scopeIDs)
				if err != nil {
					t.Fatalf("Unexpected error retrieving scopes: %s", err.Error())
				}
				result, ok := res[expectation.ID]
				if !ok {
					t.Fatalf("Expected scope %q to be in Storer, wasn't", expectation.ID)
				}

				if diff := cmp.Diff(expectation, result); diff != "" {
					t.Errorf("Unexpected result for updated scope:\n%s", diff)
				}

				for _, expected := range throwaways {
					result, ok := res[expected.ID]
					if !ok {
						t.Fatalf("Expected throwaway scope %q to be in Storer, wasn't", expected.ID)
					}
					if diff := cmp.Diff(expected, result); diff != "" {
						t.Errorf("Unexpected result for throwaway scope %q\n%s", expected.ID, diff)
					}
				}
			})
		}
	})
}

func TestUpdateNonExistent(t *testing.T) {
	// updating a scope that doesn't exist is not an error
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		deny := scopes.PolicyDefaultDeny
		change := scopes.Change{
			UserPolicy: &deny,
		}
		err := storer.Update(ctx, "https://scopes.impractical.co/test", change)
		if err != nil {
			t.Fatalf("Unexpected error updating scope: %s", err.Error())
		}
	})
}

func TestUpdateNoChange(t *testing.T) {
	// updating a scope with an empty change should not error
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		var change scopes.Change
		err := storer.Update(ctx, "https://scopes.impractical.co/test", change)
		if err != nil {
			t.Fatalf("Unexpected error updating scope: %s", err.Error())
		}
	})
}

func TestDeleteOneOfMany(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		throwaways := []scopes.Scope{
			{
				ID:         "https://scopes.impractical.co/test",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: false,
			},
			{
				ID:         "https://scopes.impractical.co/test2",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: true,
			},
			{
				ID:         "https://scopes.impractical.co/test3",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: false,
			},
			{
				ID:         "https://scopes.impractical.co/test4",
				UserPolicy: "DEFAULT_DENY",
				UserExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
					uuidOrFail(t),
				},
				ClientPolicy: "DEFAULT_DENY",
				ClientExceptions: pqarrays.StringArray{
					uuidOrFail(t),
					uuidOrFail(t),
				},
				IsDefault: true,
			},
		}
		var ids []string
		for _, scope := range throwaways {
			err := storer.Create(ctx, scope)
			if err != nil {
				t.Fatalf("Unexpected error creating scope %q: %s", scope.ID, err.Error())
			}
			ids = append(ids, scope.ID)
		}
		scope := scopes.Scope{
			ID:         "https://scopes.impractical.co/delete-me",
			UserPolicy: "DEFAULT_DENY",
			UserExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
				uuidOrFail(t),
			},
			ClientPolicy: "DEFAULT_DENY",
			ClientExceptions: pqarrays.StringArray{
				uuidOrFail(t),
				uuidOrFail(t),
			},
			IsDefault: true,
		}
		err := storer.Create(ctx, scope)
		if err != nil {
			t.Fatalf("Unexpected error creating scope %q: %s", scope.ID, err.Error())
		}
		ids = append(ids, scope.ID)

		err = storer.Delete(ctx, scope.ID)
		if err != nil {
			t.Fatalf("Unexpected error deleting scope: %s", err.Error())
		}

		res, err := storer.GetMulti(ctx, ids)
		if err != nil {
			t.Fatalf("Unexpected error retrieving scopes: %s", err.Error())
		}
		if _, ok := res[scope.ID]; ok {
			t.Errorf("Expected scope %q to not be in results, but was", scope.ID)
		}

		for _, expected := range throwaways {
			result, ok := res[expected.ID]
			if !ok {
				t.Fatalf("Expected throwaway scope %q to be in results, wasn't", expected.ID)
			}
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("Unexpected result for throwaway scope %q\n%s", expected.ID, diff)
			}
		}
	})
}

func TestDeleteNonExistent(t *testing.T) {
	runTest(t, func(t *testing.T, storer scopes.Storer, ctx context.Context) {
		// we shouldn't get an error deleting a scope that doesn't exist
		err := storer.Delete(ctx, "https://scopes.impractical.co/404")
		if err != nil {
			t.Fatalf("Unexpected error deleting account: %s", err.Error())
		}
	})
}
