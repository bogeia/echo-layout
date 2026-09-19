package biz

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
)

var (
	ErrNotFound    = errors.New("user not found")
	ErrInvalidName = errors.New("name must not be blank")
)

type User struct {
	ID   string
	Name string
}

type UserRepo interface {
	Create(context.Context, User) error
	Get(context.Context, string) (*User, error)
	List(context.Context) ([]User, error)
	Update(context.Context, User) error
	Delete(context.Context, string) error
}

type Users struct{ repo UserRepo }

func NewUsers(repo UserRepo) *Users { return &Users{repo: repo} }

func (u *Users) Create(ctx context.Context, name string) (User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return User{}, ErrInvalidName
	}
	user := User{ID: rand.Text(), Name: name}
	return user, u.repo.Create(ctx, user)
}

func (u *Users) Get(ctx context.Context, id string) (*User, error) {
	user, err := u.repo.Get(ctx, id)
	if err == nil && user == nil {
		return nil, ErrNotFound
	}
	return user, err
}

func (u *Users) List(ctx context.Context) ([]User, error) { return u.repo.List(ctx) }

func (u *Users) Update(ctx context.Context, id, name string) (User, error) {
	user := User{ID: id, Name: strings.TrimSpace(name)}
	if user.Name == "" {
		return User{}, ErrInvalidName
	}
	return user, u.repo.Update(ctx, user)
}

func (u *Users) Delete(ctx context.Context, id string) error { return u.repo.Delete(ctx, id) }
