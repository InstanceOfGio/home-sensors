package storage

import (
	"context"
	"database/sql"
	"time"

	"home-sensors/internal/models"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertDevice(ctx context.Context, mac string) (models.Device, error) {
	now := time.Now().UnixMilli()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO devices (mac, name, room, type, created_at)
		VALUES (?, ?, '', 'ht', ?)
		ON CONFLICT(mac) DO NOTHING
	`, mac, mac, now)
	if err != nil {
		return models.Device{}, err
	}

	return s.deviceByMac(ctx, mac)
}

func (s *Store) deviceByMac(ctx context.Context, mac string) (models.Device, error) {
	var d models.Device
	var createdAtMs int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id, mac, name, room, type, created_at FROM devices WHERE mac = ?
	`, mac).Scan(&d.ID, &d.Mac, &d.Name, &d.Room, &d.Type, &createdAtMs)
	if err != nil {
		return models.Device{}, err
	}
	d.CreatedAt = time.UnixMilli(createdAtMs)
	return d, nil
}

func (s *Store) InsertReading(ctx context.Context, deviceID int64, temperature, humidity float64, battery, rssi int64, ts time.Time) error {
	tsMs := ts.UnixMilli()
	nowMs := time.Now().UnixMilli()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO temperature_events (device_id, temperature, humidity, battery, rssi, created_at, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, deviceID, temperature, humidity, battery, rssi, tsMs, nowMs)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO current_state (device_id, temperature, humidity, battery, rssi, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id) DO UPDATE SET
			temperature = excluded.temperature,
			humidity = excluded.humidity,
			battery = excluded.battery,
			rssi = excluded.rssi,
			updated_at = excluded.updated_at
	`, deviceID, temperature, humidity, battery, rssi, tsMs)
	return err
}

func (s *Store) ListDevices(ctx context.Context) ([]models.Device, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, mac, name, room, type, created_at FROM devices ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.Device{}
	for rows.Next() {
		var d models.Device
		var createdAtMs int64
		if err := rows.Scan(&d.ID, &d.Mac, &d.Name, &d.Room, &d.Type, &createdAtMs); err != nil {
			return nil, err
		}
		d.CreatedAt = time.UnixMilli(createdAtMs)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) UpdateDevice(ctx context.Context, id int64, name, room, deviceType string) (models.Device, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE devices SET name = ?, room = ?, type = ? WHERE id = ?
	`, name, room, deviceType, id)
	if err != nil {
		return models.Device{}, err
	}

	var d models.Device
	var createdAtMs int64
	err = s.db.QueryRowContext(ctx, `
		SELECT id, mac, name, room, type, created_at FROM devices WHERE id = ?
	`, id).Scan(&d.ID, &d.Mac, &d.Name, &d.Room, &d.Type, &createdAtMs)
	if err != nil {
		return models.Device{}, err
	}
	d.CreatedAt = time.UnixMilli(createdAtMs)
	return d, nil
}

func (s *Store) CurrentState(ctx context.Context) ([]models.CurrentState, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.mac, d.name, d.room, cs.temperature, cs.humidity, cs.battery, cs.rssi, cs.updated_at
		FROM current_state cs
		JOIN devices d ON d.id = cs.device_id
		ORDER BY d.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.CurrentState{}
	for rows.Next() {
		var c models.CurrentState
		var updatedAtMs int64
		if err := rows.Scan(&c.DeviceID, &c.Mac, &c.Name, &c.Room, &c.Temperature, &c.Humidity, &c.Battery, &c.Rssi, &updatedAtMs); err != nil {
			return nil, err
		}
		c.UpdatedAt = time.UnixMilli(updatedAtMs)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) History(ctx context.Context, since time.Time) ([]models.HistoryPoint, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_id, created_at, temperature, humidity, battery
		FROM temperature_events
		WHERE created_at >= ?
		ORDER BY created_at ASC
	`, since.UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []models.HistoryPoint{}
	for rows.Next() {
		var p models.HistoryPoint
		var tMs int64
		if err := rows.Scan(&p.DeviceID, &tMs, &p.Temperature, &p.Humidity, &p.Battery); err != nil {
			return nil, err
		}
		p.Time = time.UnixMilli(tMs)
		out = append(out, p)
	}
	return out, rows.Err()
}
