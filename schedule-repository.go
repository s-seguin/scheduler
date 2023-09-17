package main

import (
	"database/sql"
)

type ScheduleRepository interface {
	FindAll() ([]*Schedule, error)
	FindById(id string) (*Schedule, error)
	FindByName(name string) (*Schedule, error)
	Store(schedule *Schedule) error
	Update(schedule *Schedule) error
	Delete(schedule *Schedule) error
}

type SQLScheduleRepository struct {
	DB *sql.DB
}

func NewSQLScheduleRepository(db *sql.DB) ScheduleRepository {
	return &SQLScheduleRepository{
		DB: db,
	}
}

func (r *SQLScheduleRepository) FindAll() ([]*Schedule, error) {
	return nil, nil
}

func (r *SQLScheduleRepository) FindById(id string) (*Schedule, error) {
	return nil, nil
}

func (r *SQLScheduleRepository) FindByName(name string) (*Schedule, error) {
	return nil, nil
}

func (r *SQLScheduleRepository) Store(schedule *Schedule) error {
	return nil
}

func (r *SQLScheduleRepository) Update(schedule *Schedule) error {
	return nil
}

func (r *SQLScheduleRepository) Delete(schedule *Schedule) error {
	return nil
}
