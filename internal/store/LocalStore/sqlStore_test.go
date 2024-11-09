package localstore_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	store "github.com/FonovAD/PacketSleuth/internal/store"
	localstore "github.com/FonovAD/PacketSleuth/internal/store/LocalStore"
	_ "github.com/mattn/go-sqlite3"
)

const TEST_DB_NAME = "Mydb"

func setupTestDB(t *testing.T) (*localstore.SqlStore, func()) {
	t.Helper()
	st := localstore.InitSqlStore(TEST_DB_NAME)
	return st.(*localstore.SqlStore), func() {
		st.(*localstore.SqlStore).DB.Exec("TRUNCATE TABLE " + localstore.TABLE_NAME)
		st.(*localstore.SqlStore).DB.Close()
		os.Remove(TEST_DB_NAME)
	}
}

func TestCreateTable(t *testing.T) {
	st, cleenUp := setupTestDB(t)
	defer cleenUp()

	point := store.NewPoint(time.Now(), map[string]interface{}{
		"field1": 42,
		"field2": "test",
	})

	err := st.CreateTable(*point)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	var tableName string
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	err = st.DB.QueryRow(query, localstore.TABLE_NAME).Scan(&tableName)
	if err != nil || tableName != localstore.TABLE_NAME {
		t.Fatalf("Table not created or found: %v", err)
	}
}

func TestWritePoint(t *testing.T) {
	st, cleenUp := setupTestDB(t)
	defer cleenUp()

	// Создаем таблицу
	point := store.NewPoint(time.Now(), map[string]interface{}{
		"field1": 42,
		"field2": "test",
	})
	err := st.CreateTable(*point)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Тестируем запись точки
	err = st.WritePoint(*point)
	if err != nil {
		t.Fatalf("Failed to write point: %v", err)
	}

	// Проверяем запись
	var field1 int
	var field2 string
	query := "SELECT field1, field2 FROM " + localstore.TABLE_NAME + " LIMIT 1"
	err = st.DB.QueryRow(query).Scan(&field1, &field2)
	if err != nil {
		t.Fatalf("Failed to query written point: %v", err)
	}

	// Проверяем значения
	if field1 != 42 || field2 != "test" {
		t.Fatalf("Expected field1=42 and field2='test', got field1=%v and field2='%v'", field1, field2)
	}
}

func TestDeletePoint(t *testing.T) {
	st, cleenUp := setupTestDB(t)
	defer cleenUp()

	// Создаем таблицу и добавляем точку
	point := store.NewPoint(time.Now().Truncate(time.Millisecond), map[string]interface{}{
		"field1": 42,
		"field2": "test",
	})
	err := st.CreateTable(*point)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	err = st.WritePoint(*point)
	if err != nil {
		t.Fatalf("Failed to write point: %v", err)
	}

	// Тестируем удаление точки
	err = st.DeletePoint(point.GetTimeStamp().Truncate(time.Millisecond))
	if err != nil {
		t.Fatalf("Failed to delete point: %v", err)
	}

	// Проверяем, что запись удалена
	var count int
	query := "SELECT COUNT(*) FROM " + localstore.TABLE_NAME
	err = st.DB.QueryRow(query).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query table: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected 0 rows, got %d", count)
	}
}

func TestGetPointOfTime(t *testing.T) {
	st, cleenUp := setupTestDB(t)
	defer cleenUp()

	// Создаем таблицу и добавляем несколько точек
	now := time.Now()
	points := []store.Point{
		*store.NewPoint(now.Add(-time.Hour), map[string]interface{}{"field1": 1, "field2": "one"}),
		*store.NewPoint(now, map[string]interface{}{"field1": 2, "field2": "two"}),
		*store.NewPoint(now.Add(time.Hour), map[string]interface{}{"field1": 3, "field2": "three"}),
	}
	st.CreateTable(points[0])
	for _, p := range points {
		st.WritePoint(p)
	}

	result := st.GetPointOfTime(now.Add(-30*time.Minute), now.Add(30*time.Minute))
	if len(result) != 1 {
		t.Fatalf("Expected 1 point, got %d", len(result))
	}

	if result[0].GetValue()["field2"] != "two" {
		t.Fatalf("Expected field2='two', got %v", result[0].GetValue()["field2"])
	}
}

func TestGetPointOfValue(t *testing.T) {
	st, cleenUp := setupTestDB(t)
	defer cleenUp()

	// Создаем таблицу и добавляем несколько точек
	point := store.NewPoint(time.Now(), map[string]interface{}{
		"field1": 42,
		"field2": "test_value",
	})
	err := st.CreateTable(*point)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	err = st.WritePoint(*point)
	if err != nil {
		t.Fatalf("Failed to write point: %v", err)
	}

	result := st.GetPointOfValue("field2", "test_value")
	fmt.Println(result)

	if len(result) != 1 {
		t.Fatalf("Expected 1 point, got %d", len(result))
	}
	fmt.Printf("%d, %T", result[0].GetValue()["field1"], result[0].GetValue()["field1"])
	if result[0].GetValue()["field1"] != int64(42) {
		t.Fatalf("Expected field1=42, got %v", result[0].GetValue()["field1"])
	}
}
