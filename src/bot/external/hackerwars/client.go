package hackerwars

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	SetTargetKey  = "466F2FFC"
	TeamBudgetKey = "6CDF1466"
	baseURL       = "https://aegis.hackerwars.ru"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

type apiError struct {
	Error string `json:"error"`
}

type Budget struct {
	Team string `json:"team"`
	Info struct {
		Budget uint64 `json:"budget"`
		Contributions struct {
			Today uint64 `json:"today"`
			Total uint64 `json:"total"`
		} `json:"contributions"`
		Income struct {
			Today uint64 `json:"today"`
			Total uint64 `json:"total"`
		} `json:"income"`
	} `json:"info"`
}

func SetTarget(token string, source string, target uint) error {
	values := url.Values{}
	values.Set("key", SetTargetKey)
	values.Set("token", token)
	values.Set("source", source)
	values.Set("target", strconv.FormatUint(uint64(target), 10))

	_, err := doGet("/set_target/", values)
	return err
}

func GetTeamBudget(token string) (*Budget, error) {
	values := url.Values{}
	values.Set("key", TeamBudgetKey)
	values.Set("token", token)

	body, err := doGet("/team_budget/", values)
	if err != nil {
		return nil, err
	}

	budget := Budget{}
	if err := json.Unmarshal(body, &budget); err != nil {
		return nil, fmt.Errorf("hackerwars team_budget: invalid response")
	}

	return &budget, nil
}

func doGet(path string, values url.Values) ([]byte, error) {
	requestUrl := baseURL + path + "?" + values.Encode()

	resp, err := httpClient.Get(requestUrl)
	if err != nil {
		return nil, fmt.Errorf("hackerwars request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hackerwars response read failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		parsedError := apiError{}
		if err := json.Unmarshal(body, &parsedError); err == nil && parsedError.Error != "" {
			return nil, fmt.Errorf("hackerwars error: %s", parsedError.Error)
		}

		return nil, fmt.Errorf("hackerwars request failed with status %d", resp.StatusCode)
	}

	return body, nil
}
