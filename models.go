package main

import (
	"time"
)

// Todo ...
type Todo struct {
	ID        int       `storm:"id,increment"`
	Done      bool      `storm:"index"`
	Title     string    `storm:"index"`
	CreatedAt time.Time `storm:"index"`
	UpdatedAt time.Time `storm:"index"`
}

func NewTodo(title string) *Todo {
	return &Todo{
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (t *Todo) SetTitle(title string) {
	t.Title = title
	t.UpdatedAt = time.Now()
}

func (t *Todo) ToggleDone() {
	t.Done = !t.Done
	t.UpdatedAt = time.Now()
}
