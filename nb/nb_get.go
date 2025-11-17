package nb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type NetBoxResponse struct {
	Count   int `json:"count"`
	Results []struct {
		Name     string `json:"name"`
		Platform *struct {
			Name string `json:"name"`
		} `json:"platform"`
	} `json:"results"`
}

func GetDevicePlatform(deviceName, netboxToken string) (string, error) {
	url := fmt.Sprintf("https://netbox.rt.ru/api/dcim/devices/?name=%s", deviceName)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Token "+netboxToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GoGitLabParser/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data NetBoxResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if data.Count == 0 || len(data.Results) == 0 {
		return "", nil
	}

	device := data.Results[0]
	if device.Platform != nil {
		return device.Platform.Name, nil
	}
	return "", nil
}
