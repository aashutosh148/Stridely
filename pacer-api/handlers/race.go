package handlers

import (
	"encoding/xml"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourname/pacer-api/db"
	"github.com/yourname/pacer-api/services"
)

type RaceHandler struct {
	db      *db.Postgres
	raceSvc *services.RaceService
}

func NewRaceHandler(database *db.Postgres, raceSvc *services.RaceService) *RaceHandler {
	return &RaceHandler{db: database, raceSvc: raceSvc}
}

type raceStrategyRequest struct {
	RaceName  string `json:"race_name"`
	RaceDate  string `json:"race_date"`
	GoalTimeS int    `json:"goal_time_s"`
	GPXData   string `json:"gpx_data"`
}

type splitRow struct {
	KM           int     `json:"km"`
	PaceS        int     `json:"pace_s"`
	PaceLabel    string  `json:"pace_label"`
	ElevationAdj string  `json:"elevation_adj"`
	ElevationM   float64 `json:"elevation_m"`
}

type elevationSample struct {
	distanceM float64
	elevation float64
}

type weatherSummary struct {
	TempC     float64 `json:"temp_c"`
	Condition string  `json:"condition"`
	WindKph   float64 `json:"wind_kph"`
	Humidity  int     `json:"humidity"`
}

type fuelingStep struct {
	KM     int    `json:"km"`
	Action string `json:"action"`
}

type gpxTrackPoint struct {
	Lat float64 `xml:"lat,attr"`
	Lon float64 `xml:"lon,attr"`
	Ele float64 `xml:"ele"`
}

type gpxTrackSegment struct {
	Points []gpxTrackPoint `xml:"trkpt"`
}

type gpxTrack struct {
	Segments []gpxTrackSegment `xml:"trkseg"`
}

type gpx struct {
	Tracks []gpxTrack `xml:"trk"`
}

func (h *RaceHandler) Strategy(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid user ID"})
	}

	var req raceStrategyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid body"})
	}

	if strings.TrimSpace(req.GPXData) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "gpx_data is required"})
	}

	goalPace := 300
	if req.GoalTimeS > 0 {
		goalPace = max(240, req.GoalTimeS/42)
	}

	points := parseGPXPoints(req.GPXData)
	splits := buildSplits(points, goalPace)

	weather := weatherSummary{TempC: 16, Condition: "Partly cloudy", WindKph: 9, Humidity: 58}
	if req.RaceDate != "" {
		if d, err := time.Parse("2006-01-02", req.RaceDate); err == nil {
			dayOfYear := d.YearDay()
			weather.TempC = 10 + float64(dayOfYear%18)
			weather.WindKph = 6 + float64(dayOfYear%12)
			weather.Humidity = 45 + (dayOfYear % 35)
		}
	}

	fueling := buildFuelingTimeline(len(splits))

	if len(splits) == 0 {
		for km := 1; km <= 42; km++ {
			splits = append(splits, splitRow{
				KM:           km,
				PaceS:        goalPace,
				PaceLabel:    paceLabel(goalPace),
				ElevationAdj: "0s",
				ElevationM:   0,
			})
		}
	}

	course := h.raceSvc.AnalyzeCourse(req.GPXData)

	return c.JSON(fiber.Map{
		"race_name": req.RaceName,
		"race_date": req.RaceDate,
		"goal_time_s": req.GoalTimeS,
		"course": fiber.Map{
			"difficulty": course["difficulty"],
			"key_hills":  course["key_hills"],
		},
		"weather": weather,
		"fueling_timeline": fueling,
		"splits": splits,
		"generated_for_user_id": uid,
	})
}

func (h *RaceHandler) Weather(c *fiber.Ctx) error {
	lat := c.QueryFloat("lat", 0)
	lng := c.QueryFloat("lng", 0)
	date := c.Query("date", time.Now().Format("2006-01-02"))
	return c.JSON(h.raceSvc.GetWeatherForecast(lat, lng, date))
}

func (h *RaceHandler) Fueling(c *fiber.Ctx) error {
	return c.JSON(h.raceSvc.GenerateFuelingPlan(30.0, 0.7))
}

func parseGPXPoints(raw string) []gpxTrackPoint {
	var doc gpx
	if err := xml.Unmarshal([]byte(raw), &doc); err != nil {
		return nil
	}

	points := make([]gpxTrackPoint, 0, 4096)
	for _, trk := range doc.Tracks {
		for _, seg := range trk.Segments {
			points = append(points, seg.Points...)
		}
	}
	return points
}

func buildSplits(points []gpxTrackPoint, basePace int) []splitRow {
	if len(points) < 2 {
		return nil
	}

	samples := make([]elevationSample, 0, len(points))
	totalDistance := 0.0
	prev := points[0]
	samples = append(samples, elevationSample{distanceM: totalDistance, elevation: prev.Ele})

	for i := 1; i < len(points); i++ {
		curr := points[i]
		totalDistance += haversineMeters(prev.Lat, prev.Lon, curr.Lat, curr.Lon)
		samples = append(samples, elevationSample{distanceM: totalDistance, elevation: curr.Ele})
		prev = curr
	}

	if totalDistance < 1000 {
		return nil
	}

	maxKM := int(math.Min(42, math.Floor(totalDistance/1000)))
	out := make([]splitRow, 0, maxKM)

	for km := 1; km <= maxKM; km++ {
		at := float64(km) * 1000
		elevAtKM := interpolatedElevation(samples, at)
		elevPrev := interpolatedElevation(samples, at-1000)
		delta := elevAtKM - elevPrev
		adj := int(math.Round(delta * 0.6))
		pace := max(210, basePace+adj)
		out = append(out, splitRow{
			KM:           km,
			PaceS:        pace,
			PaceLabel:    paceLabel(pace),
			ElevationAdj: elevationAdjLabel(adj),
			ElevationM:   math.Round(delta*10) / 10,
		})
	}

	return out
}

func buildFuelingTimeline(distanceKM int) []fuelingStep {
	if distanceKM <= 0 {
		distanceKM = 42
	}
	stops := []int{5, 10, 15, 20, 25, 30, 35, 40}
	steps := make([]fuelingStep, 0, len(stops))
	for i, km := range stops {
		if km > distanceKM {
			break
		}
		action := "Water"
		if i%2 == 1 {
			action = "Gel + Water"
		}
		if km >= 30 {
			action = action + " + Electrolytes"
		}
		steps = append(steps, fuelingStep{KM: km, Action: action})
	}
	sort.Slice(steps, func(i, j int) bool { return steps[i].KM < steps[j].KM })
	return steps
}

func interpolatedElevation(samples []elevationSample, target float64) float64 {
	if target <= 0 {
		return samples[0].elevation
	}
	if target >= samples[len(samples)-1].distanceM {
		return samples[len(samples)-1].elevation
	}

	for i := 1; i < len(samples); i++ {
		if samples[i].distanceM >= target {
			prev := samples[i-1]
			next := samples[i]
			span := next.distanceM - prev.distanceM
			if span <= 0 {
				return next.elevation
			}
			ratio := (target - prev.distanceM) / span
			return prev.elevation + ((next.elevation - prev.elevation) * ratio)
		}
	}

	return samples[len(samples)-1].elevation
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371000.0
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}

func degreesToRadians(v float64) float64 {
	return v * math.Pi / 180
}

func paceLabel(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	mins := seconds / 60
	secs := seconds % 60
	return formatInt(mins) + ":" + twoDigits(secs)
}

func elevationAdjLabel(adj int) string {
	if adj == 0 {
		return "0s"
	}
	if adj > 0 {
		return "+" + formatInt(adj) + "s"
	}
	return formatInt(adj) + "s"
}

func twoDigits(v int) string {
	if v < 10 {
		return "0" + formatInt(v)
	}
	return formatInt(v)
}

func formatInt(v int) string {
	return strconv.Itoa(v)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
