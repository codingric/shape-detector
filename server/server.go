package server

import (
	"encoding/json"
	"net/http"

	"github.com/codingric/shape-detector/service"
	"github.com/rs/zerolog/log"
)

type AnalyzeRequest struct {
	Url   string        `json:"url"`
	Zones []AnalyzeZone `json:"zones"`
}

type AnalyzeZone struct {
	Coords    [4]int `json:"coords"`
	Name      string `json:"name"`
	Threshold int    `json:"threshold"`
}

type AnalyzeResponse struct {
	Detections map[string]bool `json:"detections"`
	Image      string          `json:"image"`
}

func AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	submitted := AnalyzeRequest{}
	err := json.NewDecoder(r.Body).Decode(&submitted)
	if err != nil {
		log.Error().Err(err).Msg("Request error")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Info().Interface("request", submitted).Msg("Request decoded")

	zones := make([]service.IAZone, len(submitted.Zones))
	for i, zone := range submitted.Zones {
		zones[i] = service.IAZone{
			X1:        zone.Coords[0],
			Y1:        zone.Coords[1],
			X2:        zone.Coords[2],
			Y2:        zone.Coords[3],
			Name:      zone.Name,
			Threshold: zone.Threshold,
		}
	}
	svc := service.NewImageService(r.Context(), submitted.Url, zones...)
	detections, err := svc.Analyze()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	response := AnalyzeResponse{Detections: detections, Image: svc.Base64()}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Info().Interface("response", response).Msg("Response sent")
}

func StartServer(port string) error {
	http.HandleFunc("/analyze", AnalyzeHandler)
	log.Info().Str("port", port).Msg("Started server")
	return http.ListenAndServe(":"+port, nil)
}
