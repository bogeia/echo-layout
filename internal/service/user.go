package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/bogeia/echo-layout/internal/biz"
	"github.com/danielgtaylor/huma/v2"
)

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserBody struct {
	Name string `json:"name" minLength:"1" maxLength:"100" doc:"用户名称，不能仅包含空白"`
}

type (
	CreateUserInput struct{ Body UserBody }
	UserIDInput     struct {
		ID string `path:"id" minLength:"1" maxLength:"128"`
	}
)

type UpdateUserInput struct {
	ID   string `path:"id" minLength:"1" maxLength:"128"`
	Body UserBody
}
type (
	UserOutput  struct{ Body User }
	UsersOutput struct{ Body []User }
)

type Users struct {
	biz *biz.Users
	log *slog.Logger
}

func NewUsers(biz *biz.Users, log *slog.Logger) *Users { return &Users{biz: biz, log: log} }

func (u *Users) Create(ctx context.Context, input *CreateUserInput) (*UserOutput, error) {
	user, err := u.biz.Create(ctx, input.Body.Name)
	if err != nil {
		return nil, u.apiError(ctx, err)
	}
	return &UserOutput{Body: User{ID: user.ID, Name: user.Name}}, nil
}

func (u *Users) Get(ctx context.Context, input *UserIDInput) (*UserOutput, error) {
	user, err := u.biz.Get(ctx, input.ID)
	if err != nil {
		return nil, u.apiError(ctx, err)
	}
	return &UserOutput{Body: User{ID: user.ID, Name: user.Name}}, nil
}

func (u *Users) List(ctx context.Context, _ *struct{}) (*UsersOutput, error) {
	users, err := u.biz.List(ctx)
	if err != nil {
		return nil, u.apiError(ctx, err)
	}
	result := make([]User, 0, len(users))
	for _, user := range users {
		result = append(result, User{ID: user.ID, Name: user.Name})
	}
	return &UsersOutput{Body: result}, nil
}

func (u *Users) Update(ctx context.Context, input *UpdateUserInput) (*UserOutput, error) {
	user, err := u.biz.Update(ctx, input.ID, input.Body.Name)
	if err != nil {
		return nil, u.apiError(ctx, err)
	}
	return &UserOutput{Body: User{ID: user.ID, Name: user.Name}}, nil
}

func (u *Users) Delete(ctx context.Context, input *UserIDInput) (*struct{}, error) {
	if err := u.biz.Delete(ctx, input.ID); err != nil {
		return nil, u.apiError(ctx, err)
	}
	return nil, nil
}

func (u *Users) apiError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, biz.ErrNotFound):
		return huma.Error404NotFound(err.Error())
	case errors.Is(err, biz.ErrInvalidName):
		return huma.Error422UnprocessableEntity(err.Error())
	default:
		u.log.ErrorContext(ctx, "internal error", "error", err)
		return huma.Error500InternalServerError("internal server error")
	}
}
