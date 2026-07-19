package models

import "time"

type Device struct {
	ID        int64     `json:"id"`
	Mac       string    `json:"mac"`
	Name      string    `json:"name"`
	Room      string    `json:"room"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type CurrentState struct {
	DeviceID    int64     `json:"device_id"`
	Mac         string    `json:"mac"`
	Name        string    `json:"name"`
	Room        string    `json:"room"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Battery     int64     `json:"battery"`
	Rssi        int64     `json:"rssi"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type HistoryPoint struct {
	DeviceID    int64     `json:"device_id"`
	Time        time.Time `json:"t"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Battery     int64     `json:"battery"`
}

type WeatherPoint struct {
	Time                     time.Time `json:"t"`
	Temperature              float64   `json:"temperature"`
	PrecipitationProbability float64   `json:"precipitation_probability"`
	WindSpeed                float64   `json:"wind_speed"`
	Humidity                 float64   `json:"humidity"`
}

type MinewPayload struct {
	Tm  time.Time  `json:"tm"`
	Gw  string     `json:"gw"`
	Seq int64      `json:"seq"`
	Adv []MinewAdv `json:"adv"`
}

type MinewAdv struct {
	Type        string    `json:"type"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Battery     int64     `json:"battery"`
	Rssi        int64     `json:"rssi"`
	Tm          time.Time `json:"tm"`
	Mac         string    `json:"mac"`
}
