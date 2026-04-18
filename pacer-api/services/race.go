package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type RaceService struct{}

func NewRaceService() *RaceService {
	return &RaceService{}
}

func (s *RaceService) AnalyzeCourse(gpxData string) map[string]interface{} {
	// Dummy GPX parsing logic
	difficulty := 0.5
	keyHills := []string{}
	
	if strings.Contains(gpxData, "<ele>") {
		difficulty = 0.8
		keyHills = append(keyHills, "Extracted Hill (gradient > 3%, length > 200m)")
	}

	return map[string]interface{}{
		"difficulty": difficulty,
		"key_hills":  keyHills,
	}
}

func (s *RaceService) GetWeatherForecast(lat float64, lng float64, date string) map[string]interface{} {
	apiKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	if apiKey == "" {
		return map[string]interface{}{
			"temp": 15.0,
			"error": "OPENWEATHERMAP_API_KEY not set, using default data",
		}
	}
	
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?lat=%f&lon=%f&appid=%s&units=metric", lat, lng, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	defer resp.Body.Close()
	
	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)
	return data
}

func (s *RaceService) GeneratePacingStrategy(course map[string]interface{}, weather map[string]interface{}, fitness float64) []map[string]interface{} {
	// km-by-km splits from course + weather + fitness
	return []map[string]interface{}{
		{"km": 1, "pace": "5:00", "elevation_adj": "0s"},
		{"km": 2, "pace": "4:55", "elevation_adj": "-5s"},
		{"km": 3, "pace": "5:10", "elevation_adj": "+10s (hill)"},
	}
}

func (s *RaceService) GenerateFuelingPlan(wallDistance float64, sweatRate float64) map[string]interface{} {
	// aid station plan from wall prediction + sweat rate
	return map[string]interface{}{
		"total_gels": 4,
		"water_ml_per_hour": sweatRate * 1000,
		"stations": []map[string]interface{}{
			{"km": 5, "action": "Water"},
			{"km": 10, "action": "Gel + Water"},
		},
	}
}
