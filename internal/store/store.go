package store

import (
	"errors"
	"sync"
	"time"
)

type Store interface {
	//Create a table with the fields passed in the structure
	CreateTable(Point) error
	WritePoint(Point) error
	DeletePoint(time.Time) error
	GetPointOfTime(time.Time, time.Time) []Point
	GetPointOfValue(string, interface{}) []Point
	Close() error
}

type Point struct {
	timestamp time.Time
	values    map[string]interface{}
}

func NewPoint(ts time.Time, v map[string]interface{}) *Point {
	return &Point{
		timestamp: ts,
		values:    v,
	}
}

func (p *Point) GetTimeStamp() time.Time {
	return p.timestamp
}

func (p *Point) GetValue() map[string]interface{} {
	return p.values
}

type MockStore struct {
	DB map[time.Time]map[string]interface{}
	mu sync.Mutex
}

func InitMockStore() *MockStore {
	db := make(map[time.Time]map[string]interface{})
	return &MockStore{
		DB: db,
	}
}

func (s *MockStore) CreateTable(p Point) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil
}

func (s *MockStore) Close() error {
	return nil
}

func (s *MockStore) WritePoint(p Point) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DB[p.timestamp] = p.values
	return nil
}

func (s *MockStore) DeletePoint(ts time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, in := s.DB[ts]
	delete(s.DB, ts)
	_, inAfterDel := s.DB[ts]
	if in && !inAfterDel {
		return nil
	}
	return errors.New("delete point error")
}

func (s *MockStore) GetPointOfTime(tsFrom, tsTo time.Time) []Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	points := []Point{}
	for ts, v := range s.DB {
		if ts.After(tsFrom) && ts.Before(tsTo) {
			points = append(points, Point{
				timestamp: ts,
				values:    v,
			})
		}
	}
	return points
}

func (s *MockStore) GetPointOfValue(key string, value interface{}) []Point {
	s.mu.Lock()
	defer s.mu.Unlock()
	points := []Point{}
	for ts, v := range s.DB {
		if s.DB[ts][key] == value {
			points = append(points, Point{
				timestamp: ts,
				values:    v,
			})
		}
	}
	return points
}
