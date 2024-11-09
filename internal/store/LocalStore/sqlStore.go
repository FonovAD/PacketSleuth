package localstore

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/FonovAD/PacketSleuth/internal/store"
)

const (
	TYPE_INT    = "int"
	TYPE_STRING = "string"
	TYPE_FLOAT  = "float"
	TYPE_BOOL   = "bool"

	SQLTYPE_INT    = "INTEGER"
	SQLTYPE_STRING = "TEXT"
	SQLTYPE_FLOAT  = "NUMERIC"
	SQLTYPE_BOOL   = "BOOL"

	TABLE_NAME = "packet_sleuth"
)

type SqlStore struct {
	DB *sql.DB
}

// в дальнейшем нужно будет переделать метод:
// - вынести подключение к бд в отдельнею функию
// - поменять параметр dbName на db sql.DB
// тогда можно будет использовать любую sql бд, а не только sqlite
func InitSqlStore(dbName string) store.Store {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		panic(err)
	}
	return &SqlStore{
		DB: db,
	}
}

func (s *SqlStore) CreateTable(p store.Point) error {
	attributes := []string{}
	attributes = append(attributes, "ts DATETIME")
	for i, j := range p.GetValue() {
		attribute := i
		switch fmt.Sprintf("%T", j) {
		case TYPE_INT:
			attribute += " " + SQLTYPE_INT
		case TYPE_STRING:
			attribute += " " + SQLTYPE_STRING
		case TYPE_FLOAT:
			attribute += " " + SQLTYPE_FLOAT
		case TYPE_BOOL:
			attribute += " " + SQLTYPE_BOOL
		}
		attributes = append(attributes, attribute)
	}
	SQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", TABLE_NAME)
	for i := range len(attributes) {
		if i != len(attributes)-1 {
			SQL += attributes[i] + ", "
		} else {
			SQL += attributes[i] + ")"
		}
	}
	fmt.Println(SQL)
	_, err := s.DB.Exec(SQL)
	if err != nil {
		return err
	}
	return nil
}

// нужно добавить цикл, в котором название полей и значения будут
// записываться в два массива. Это нужно чтобы они не перемешались,
// т.к. в Go в типе map значения могут быть перемешаны
func (s *SqlStore) WritePoint(p store.Point) error {
	SQL := fmt.Sprintf("INSERT INTO %s (ts, ", TABLE_NAME)
	for i := range p.GetValue() {
		SQL += i + ", "
	}
	SQL = SQL[0:len(SQL)-2] + ") VALUES ("
	values := []interface{}{}
	values = append(values, p.GetTimeStamp())
	i := 1
	for _, j := range p.GetValue() {
		SQL += fmt.Sprintf("$%d, ", i)
		i += 1
		values = append(values, j)
	}
	SQL += "$3)"
	fmt.Println(SQL)

	_, err := s.DB.Exec(SQL, values...)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) DeletePoint(ts time.Time) error {
	SQL := fmt.Sprintf("DELETE FROM %s WHERE ts = $1", TABLE_NAME)
	fmt.Println(SQL, ts)

	_, err := s.DB.Exec(SQL, ts)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) GetPointOfTime(tsFrom, tsTo time.Time) []store.Point {
	SQL := fmt.Sprintf("SELECT * FROM %s WHERE ts > $1 and ts < $2", TABLE_NAME)
	fmt.Println(SQL)

	rows, err := s.DB.Query(SQL, tsFrom, tsTo)
	if err != nil {
		fmt.Printf("Failed to execute query: %v\n", err)
		return []store.Point{}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		fmt.Printf("Failed to get columns: %v\n", err)
		return []store.Point{}
	}

	ps := []store.Point{}
	for rows.Next() {
		var ts time.Time
		pointValues := make(map[string]interface{})
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			fmt.Printf("Failed to scan row: %v\n", err)
			return []store.Point{}
		}

		for i, colName := range columns {
			val := values[i]
			if colName == "ts" {
				ts = val.(time.Time)
			} else {
				if valPtr, ok := val.(*interface{}); ok {
					pointValues[colName] = *valPtr
				} else {
					pointValues[colName] = val
				}
			}
		}
		ps = append(ps, *store.NewPoint(ts, pointValues))
	}
	return ps
}

func (s *SqlStore) GetPointOfValue(key string, value interface{}) []store.Point {
	SQL := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", TABLE_NAME, key)
	fmt.Println(SQL)

	rows, err := s.DB.Query(SQL, value)
	if err != nil {
		return []store.Point{}
	}
	columns, err := rows.Columns()
	if err != nil {
		return []store.Point{}
	}
	return ParseRowsToPoints(columns, rows)
}

func ParseRowsToPoints(columns []string, rows *sql.Rows) []store.Point {
	ps := []store.Point{}
	for rows.Next() {
		var ts time.Time
		pointValues := make(map[string]interface{})

		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}

		if err := rows.Scan(valuePointers...); err != nil {
			fmt.Printf("Failed to scan row: %v\n", err)
			return []store.Point{}
		}

		for i, colName := range columns {
			val := values[i]
			if colName == "ts" {
				ts = val.(time.Time)
			} else {
				if valPtr, ok := val.(*interface{}); ok {
					pointValues[colName] = *valPtr
				} else {
					pointValues[colName] = val
				}
			}
		}
		ps = append(ps, *store.NewPoint(ts, pointValues))
	}
	return ps
}
