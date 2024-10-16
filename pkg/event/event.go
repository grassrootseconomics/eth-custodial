package event

import "encoding/json"

type (
	Event struct {
		TrackingID string `json:"trackingId"`
		Status     string `json:"status"`
	}
)

func (e Event) Serialize() ([]byte, error) {
	jsonData, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return jsonData, err
}

func Deserialize(jsonData []byte) (Event, error) {
	var (
		event Event
	)

	if err := json.Unmarshal(jsonData, &event); err != nil {
		return event, err
	}

	return event, nil
}
