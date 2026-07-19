CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mac TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    room TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'ht',
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS temperature_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL REFERENCES devices(id),
    temperature REAL NOT NULL,
    humidity REAL,
    battery INTEGER,
    rssi INTEGER,
    created_at INTEGER NOT NULL,
    received_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_temperature_events_device_time
    ON temperature_events (device_id, created_at);

CREATE TABLE IF NOT EXISTS current_state (
    device_id INTEGER PRIMARY KEY REFERENCES devices(id),
    temperature REAL,
    humidity REAL,
    battery INTEGER,
    rssi INTEGER,
    updated_at INTEGER NOT NULL
);
