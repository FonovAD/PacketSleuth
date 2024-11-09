package store_test

import (
	"testing"
	"time"

	"github.com/FonovAD/PacketSleuth/internal/store"
	"github.com/stretchr/testify/assert"
)

func TestNewPoint(t *testing.T) {
	// Инициализация времени и карты значений
	ts := time.Now()
	values := map[string]interface{}{
		"tcpCount": 34,
		"IPv4":     "192.168.0.1",
	}
	point := store.NewPoint(ts, values)
	assert.NotNil(t, point, "Point should not be nil")
	assert.Equal(t, ts, point.GetTimeStamp(), "Timestamp should match the input")
	assert.Equal(t, values, point.GetValue(), "Values map should match the input")
	assert.Equal(t, 34, point.GetValue()["tcpCount"])
	assert.Equal(t, "192.168.0.1", point.GetValue()["IPv4"])
}

func TestNewPointWithNilValues(t *testing.T) {
	ts := time.Now()
	point := store.NewPoint(ts, nil)
	assert.NotNil(t, point, "Point should not be nil")
	assert.Equal(t, ts, point.GetTimeStamp(), "Timestamp should match the input")
	assert.Nil(t, point.GetValue(), "Values should be nil")
}

func TestNewPointWithEmptyValues(t *testing.T) {
	ts := time.Now()

	point := store.NewPoint(ts, map[string]interface{}{})
	assert.NotNil(t, point, "Point should not be nil")
	assert.Equal(t, ts, point.GetTimeStamp(), "Timestamp should match the input")

	assert.NotNil(t, point.GetValue(), "Values should not be nil")
	assert.Empty(t, point.GetValue(), "Values map should be empty")
}

func TestInitMockStore(t *testing.T) {
	st := store.InitMockStore()

	assert.NotNil(t, st, "Store should not be nil")
	assert.Empty(t, st.DB, "Store db should be empty initially")
}

func TestCreateTable(t *testing.T) {
	st := store.InitMockStore()
	point := store.NewPoint(time.Now(), map[string]interface{}{"key": "value"})

	result := st.CreateTable(*point)

	assert.Equal(t, nil, result, "CreateTable should return 0")
}

func TestWritePoint(t *testing.T) {
	st := store.InitMockStore()
	point := store.NewPoint(time.Now(), map[string]interface{}{"tcpCount": 34})

	result := st.WritePoint(*point)

	assert.Equal(t, nil, result, "WritePoint should return 0")
	assert.Contains(t, st.DB, point.GetTimeStamp(), "Store should contain the timestamp")
	assert.Equal(t, st.DB[point.GetTimeStamp()], point.GetValue(), "The point values should match")
}

func TestDeletePoint(t *testing.T) {
	st := store.InitMockStore()
	point := store.NewPoint(time.Now(), map[string]interface{}{"tcpCount": 34})

	st.WritePoint(*point)
	assert.Contains(t, st.DB, point.GetTimeStamp())
	result := st.DeletePoint(point.GetTimeStamp())

	assert.Equal(t, nil, result, "DeletePoint should return 0")
	assert.NotContains(t, st.DB, point.GetTimeStamp(), "Store should not contain the deleted point")

	result = st.DeletePoint(point.GetTimeStamp())
	assert.NotEqual(t, nil, result, "DeletePoint of non-existent point should return 1")
}

func TestGetPointOfTime(t *testing.T) {
	st := store.InitMockStore()
	now := time.Now()
	point1 := store.NewPoint(now.Add(-time.Hour), map[string]interface{}{"tcpCount": 20})
	point2 := store.NewPoint(now.Add(time.Hour), map[string]interface{}{"tcpCount": 30})
	st.WritePoint(*point1)
	st.WritePoint(*point2)
	points := st.GetPointOfTime(now.Add(-2*time.Hour), now.Add(2*time.Hour))

	assert.Len(t, points, 2, "Should return two points within the time range")
	assert.Contains(t, points, *point1, "Point1 should be included in the result")
	assert.Contains(t, points, *point2, "Point2 should be included in the result")

	points = st.GetPointOfTime(now.Add(2*time.Hour), now.Add(3*time.Hour))
	assert.Empty(t, points, "Should return no points if no points are in the time range")
}

func TestGetPointOfValue_1(t *testing.T) {
	st := store.InitMockStore()

	point1 := store.NewPoint(time.Now(), map[string]interface{}{"tcpCount": 25})
	point2 := store.NewPoint(time.Now().Add(time.Hour), map[string]interface{}{"tcpCount": 30})
	st.WritePoint(*point1)
	st.WritePoint(*point2)

	points := st.GetPointOfValue("tcpCount", 25)

	assert.Len(t, points, 1, "Should return one point with the specified value")
	assert.Contains(t, points, *point1, "Point1 should be included in the result")

	points = st.GetPointOfValue("tcpCount", 35)
	assert.Empty(t, points, "Should return no points if no points match the value")
}

func TestGetPointOfValue_2(t *testing.T) {
	st := store.InitMockStore()

	point1 := store.NewPoint(time.Now(), map[string]interface{}{"tcpCount": 25})
	point2 := store.NewPoint(time.Now().Add(time.Hour), map[string]interface{}{"tcpCount": 30})
	st.WritePoint(*point1)
	st.WritePoint(*point2)

	points := st.GetPointOfValue("tcpCount", 25)

	assert.Len(t, points, 1, "Should return one point with the specified value")
	assert.Contains(t, points, *point1, "Point1 should be included in the result")

	points = st.GetPointOfValue("tcpCount", "25")
	assert.Empty(t, points, "Should return no points if no points match the value")
}
