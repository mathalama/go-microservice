package client

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DoctorHTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewDoctorHTTPClient(baseURL string, timeout time.Duration) *DoctorHTTPClient {
	return &DoctorHTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *DoctorHTTPClient) DoctorExists(doctorID string) (bool, error) {
	url := fmt.Sprintf("%s/doctors/%s", c.baseURL, doctorID)
	resp, err := c.client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected doctor-service status: %d", resp.StatusCode)
	}
}
