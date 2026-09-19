package data

import (
	"context"
	"log/slog"

	"github.com/bogeia/echo-layout/internal/biz"
)

type userRepo struct {
	data *Data
	log  *slog.Logger
}

func NewUsers(data *Data, log *slog.Logger) biz.UserRepo {
	return &userRepo{data: data, log: log}
}

// These methods intentionally do not persist data.
func (u *userRepo) Create(ctx context.Context, user biz.User) error {
	u.log.InfoContext(ctx, "data.users.create", "id", user.ID)
	return nil
}

func (u *userRepo) Get(ctx context.Context, id string) (*biz.User, error) {
	u.log.InfoContext(ctx, "data.users.get", "id", id)
	return nil, nil
}

func (u *userRepo) List(ctx context.Context) ([]biz.User, error) {
	u.log.InfoContext(ctx, "data.users.list")
	return []biz.User{}, nil
}

func (u *userRepo) Update(ctx context.Context, user biz.User) error {
	u.log.InfoContext(ctx, "data.users.update", "id", user.ID)
	return nil
}

func (u *userRepo) Delete(ctx context.Context, id string) error {
	u.log.InfoContext(ctx, "data.users.delete", "id", id)
	return nil
}
