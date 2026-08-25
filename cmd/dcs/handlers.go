package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chemicalprocessdcs/internal/model"
	"chemicalprocessdcs/internal/param"
	"chemicalprocessdcs/internal/record"
)

// notFound reports whether err is the reactor-not-found error returned when a
// reactor id has not been profiled/registered yet. The interlock scan treats
// such ids as "skip this reactor" rather than a hard failure.
func notFound(err error) bool {
	return errors.Is(err, model.ErrReactorNotFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.journal.AppendEvent(model.NewEvent("system", "health", "ping", time.Now().UTC()))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListReactors(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "list")
	count := s.reactor.ReactorCount()
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (s *Server) handleReactor(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/reactors/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing reactor id"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		if strings.HasSuffix(r.URL.Path, "/trend") {
			trendID := strings.TrimSuffix(id, "/trend")
			trend, ok := s.reactor.Trend(trendID)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "no trend"})
				return
			}
			writeJSON(w, http.StatusOK, trend)
		} else if strings.HasSuffix(r.URL.Path, "/stats") {
			statsID := strings.TrimSuffix(id, "/stats")
			stats, err := s.reactor.Statistics(statsID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, stats)
		} else if strings.HasSuffix(r.URL.Path, "/ramp") {
			rampID := strings.TrimSuffix(id, "/ramp")
			elapsed := 0
			if q := r.URL.Query().Get("elapsed"); q != "" {
				elapsed, _ = strconv.Atoi(q)
			}
			next, done, err := s.reactor.RampProgress(rampID, elapsed)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"next_temperature": next, "done": done})
		} else if strings.HasSuffix(r.URL.Path, "/recipe") {
			recipeID := strings.TrimSuffix(id, "/recipe")
			recipe, err := s.reactor.Recipe(recipeID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, recipe)
		} else if strings.HasSuffix(r.URL.Path, "/report") {
			reportID := strings.TrimSuffix(id, "/report")
			report, err := s.reactor.BatchReport(reportID)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, report)
		} else {
			status, err := s.reactor.Status(id)
			if err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, status)
		}
	case http.MethodPost:
		var body struct {
			Action      string  `json:"action"`
			Temperature float64 `json:"temperature"`
			Pressure    float64 `json:"pressure"`
			Steps       int     `json:"steps"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
			return
		}
		switch body.Action {
		case "register":
			s.reactor.Register(id)
			writeJSON(w, http.StatusOK, map[string]string{"status": "registered"})
		case "reading":
			if err := s.reactor.SetReading(id, body.Temperature, body.Pressure); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "reading set"})
		case "step":
			if err := s.reactor.Step(id); err != nil {
				if notFound(err) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "stepped"})
		case "simulate":
			if body.Steps <= 0 {
				body.Steps = 1
			}
			statuses, err := s.reactor.Simulate(id, body.Steps)
			if err != nil {
				if notFound(err) {
					writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
					return
				}
				writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, statuses)
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown action"})
		}
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "status")
	telemetry := s.reactor.Telemetry()
	writeJSON(w, http.StatusOK, map[string]any{
		"count":          telemetry.ReactorCount(),
		"statuses":       telemetry.Statuses,
		"tripped":        telemetry.TrippedIDs(),
		"alarm_summary":  s.alarm.Summary(),
		"restored_count": s.alarm.RestoredCount(),
		"interlocked":    s.esd.InterlockedCount(),
	})
}

func (s *Server) handleAlarms(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "alarms")
	if r.Method == http.MethodPost {
		var body struct {
			ReactorID string `json:"reactor_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
			return
		}
		s.alarm.Acknowledge(body.ReactorID)
		writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"active_count": s.alarm.ActiveCount(),
		"history":      s.alarm.History(),
		"summary":      s.alarm.Summary(),
	})
}

func (s *Server) handleCurves(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "curves")
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.params.AllCurves())
	case http.MethodPost:
		var c model.Curve
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
			return
		}
		if err := param.ValidateCurve(c); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.params.SetCurve(c)
		writeJSON(w, http.StatusOK, map[string]string{"status": "curve set"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleRecipes(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "recipes")
	switch r.Method {
	case http.MethodGet:
		if q := r.URL.Query().Get("id"); q != "" {
			curves, ok := s.params.RecipeCurves(q)
			if !ok {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": "no recipe"})
				return
			}
			writeJSON(w, http.StatusOK, curves)
			return
		}
		writeJSON(w, http.StatusOK, s.params.AllRecipes())
	case http.MethodPost:
		var recipe param.Recipe
		if err := json.NewDecoder(r.Body).Decode(&recipe); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
			return
		}
		s.params.SetRecipe(recipe)
		writeJSON(w, http.StatusOK, map[string]string{"status": "recipe set"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleRecords(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "records")
	if r.Method == http.MethodPost {
		var body struct {
			MaxEntries int `json:"max_entries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
			return
		}
		if err := s.record.Trim(body.MaxEntries); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "trimmed"})
		return
	}
	entries, err := s.record.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	average, peakTemp, peakPressure := record.StatisticsOf(entries)
	digest, _ := s.record.VerifyDigest()
	writeJSON(w, http.StatusOK, map[string]any{
		"entries":          entries,
		"average_temp":     average,
		"peak_temperature": peakTemp,
		"peak_pressure":    peakPressure,
		"digest":           digest,
	})
}

func (s *Server) handleJournal(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "journal")
	lines, err := s.journal.ReadAll()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines, "count": len(lines)})
}

func (s *Server) handleParams(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "params")
	switch r.Method {
	case http.MethodGet:
		params, ok := s.params.Params("main")
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no params"})
			return
		}
		writeJSON(w, http.StatusOK, params)
	case http.MethodPost:
		var p model.ProcessParams
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
			return
		}
		if err := param.ValidateParams(p); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		s.params.SetParams("main", p)
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleTrip(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "trip")
	var body struct {
		ReactorID string `json:"reactor_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	if err := s.esd.Trip(body.ReactorID); err != nil {
		s.alarm.ReportTrip(body.ReactorID, err)
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "tripped", "reason": string(s.esd.TripReason(body.ReactorID))})
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	s.journal.Append("system", "restore")
	var body struct {
		ReactorID string `json:"reactor_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request"})
		return
	}
	if err := s.alarm.Restore(body.ReactorID); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
