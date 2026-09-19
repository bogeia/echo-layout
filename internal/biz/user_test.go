package biz_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bogeia/echo-layout/internal/biz"
)

type userRepository struct {
	user    *biz.User
	err     error
	creates int
}

func (r *userRepository) Create(_ context.Context, user biz.User) error {
	r.creates++
	r.user = &user
	return r.err
}

func (r *userRepository) Get(context.Context, string) (*biz.User, error) {
	return r.user, r.err
}

func (r *userRepository) List(context.Context) ([]biz.User, error) {
	return nil, r.err
}

func (r *userRepository) Update(_ context.Context, user biz.User) error {
	r.user = &user
	return r.err
}

func (r *userRepository) Delete(context.Context, string) error {
	r.user = nil
	return r.err
}

func TestUsersWithRepository(t *testing.T) {
	repo := &userRepository{}
	users := biz.NewUsers(repo)
	ctx := t.Context()
	if _, err := users.Create(ctx, "  "); !errors.Is(err, biz.ErrInvalidName) || repo.creates != 0 {
		t.Fatal("blank name must fail before calling the repository")
	}
	created, err := users.Create(ctx, " Alice ")
	if err != nil || created.ID == "" || created.Name != "Alice" || repo.user == nil || *repo.user != created {
		t.Fatalf("create: user=%+v, error=%v", created, err)
	}
	updated, err := users.Update(ctx, created.ID, " Bob ")
	if err != nil || updated.Name != "Bob" || updated.ID != created.ID || *repo.user != updated {
		t.Fatalf("update: user=%+v, error=%v", updated, err)
	}
	if _, err := users.Update(ctx, created.ID, " "); !errors.Is(err, biz.ErrInvalidName) || *repo.user != updated {
		t.Fatal("blank update must not reach the repository")
	}
	if found, err := users.Get(ctx, created.ID); err != nil || found == nil || *found != updated {
		t.Fatalf("get: user=%+v, error=%v", found, err)
	}
	if err := users.Delete(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := users.Get(ctx, created.ID); !errors.Is(err, biz.ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}
	repo.err = errors.New("repository unavailable")
	_, createErr := users.Create(ctx, "Alice")
	_, getErr := users.Get(ctx, created.ID)
	_, listErr := users.List(ctx)
	_, updateErr := users.Update(ctx, created.ID, "Bob")
	deleteErr := users.Delete(ctx, created.ID)
	for _, err := range []error{createErr, getErr, listErr, updateErr, deleteErr} {
		if !errors.Is(err, repo.err) {
			t.Fatalf("repository error was not preserved: %v", err)
		}
	}
}
