package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"image"
	"io"
	"math"
	"net/http"
	"os"

	"github.com/disintegration/imaging"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

type ImageAnalyzer interface {
	Analyze() (map[string]bool, error)
	Download() error
	Mask() error
	Adjust() error
	SaveAs(string) error
	Base64() string
}

type IAZone struct {
	X1        int
	Y1        int
	X2        int
	Y2        int
	Name      string
	Threshold int
}

type ImageService struct {
	image image.Image
	zones []IAZone
	url   string
}

func (s *ImageService) Base64() string {
	if s.image != nil {
		data := new(bytes.Buffer)
		imaging.Encode(data, s.image, imaging.JPEG)
		return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data.Bytes())
	}
	return ""
}

func (s *ImageService) Analyze() (detections map[string]bool, err error) {
	err = s.Download()
	if err != nil {
		panic(err)
	}

	s.SaveAs("original.jpeg")
	err = s.Adjust()
	if err != nil {
		panic(err)
	}
	s.SaveAs("adjusted.jpeg")
	err = s.Mask()
	if err != nil {
		panic(err)
	}
	s.SaveAs("masked.jpeg")

	// Foreach zone, detect area of white pixles
	detections = make(map[string]bool)
	for _, zone := range s.zones {
		rect := image.Rect(zone.X1, zone.Y1, zone.X2, zone.Y2)
		cropped := imaging.Crop(s.image, rect)
		count := 0
		for x := 0; x < cropped.Bounds().Dx(); x++ {
			for y := 0; y < cropped.Bounds().Dy(); y++ {
				r, g, b, a := cropped.At(x, y).RGBA()
				ra := float64(r) / float64(a)
				ga := float64(g) / float64(a)
				ba := float64(b) / float64(a)
				br := (ra + ga + ba) / 3
				if br > 0.5 {
					log.Trace().Int("x", x).Int("y", y).Float64("level", br).Msg("White pixel detected")
					count++
				}
			}
		}
		log.Debug().Str("zone", zone.Name).Int("count", count).Msg("Zone analyzed")
		if count > zone.Threshold {
			detections[zone.Name] = true
		} else {
			detections[zone.Name] = false
		}
	}

	return
}

func (s *ImageService) Download() (err error) {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	log.Trace().Str("url", s.url).Msg("Getting image from camera URL")
	resp, err := (&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}).Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	size := len(imageData)
	log.Debug().Int("bytes", size).Msg("Image data read")
	if size < 100 {
		return fmt.Errorf("image size is %d bytes", size)
	}

	// Increase contrast and brightness to improve model interpretation
	s.image, err = imaging.Decode(bytes.NewReader(imageData))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	log.Info().Msg("Image extracted")
	return
}

func (s *ImageService) Adjust() error {
	s.image = imaging.Grayscale(s.image)
	return nil
}

func (s *ImageService) Mask() error {
	bounds := s.image.Bounds()
	mask := imaging.New(bounds.Dx(), bounds.Dy(), image.Black)
	//mask := imaging.AdjustContrast(s.image, -100)

	for _, zone := range s.zones {
		rect := image.Rect(zone.X1, zone.Y1, zone.X2, zone.Y2)
		cropped := imaging.Crop(s.image, rect)
		px := uint64(cropped.Bounds().Dx() * cropped.Bounds().Dy())
		bright := make([]float64, px)
		i := 0
		for x := 0; x < cropped.Bounds().Dx(); x++ {
			for y := 0; y < cropped.Bounds().Dy(); y++ {
				r, g, b, a := cropped.At(x, y).RGBA()
				ra := float64(r) / float64(a)
				ga := float64(g) / float64(a)
				ba := float64(b) / float64(a)
				br := (ra + ga + ba) / 3

				// Using standard luminance formula: 0.299R + 0.587G + 0.114B
				bright[i] = br
				i += 1
			}
		}
		median := Median(bright)
		cropped = imaging.AdjustSigmoid(cropped, math.Max(median, 0.3), 50)
		mask = imaging.Paste(mask, cropped, image.Pt(zone.X1, zone.Y1))
	}
	log.Info().Msg("Image masked")
	s.image = mask

	return nil
}

func (s *ImageService) SaveAs(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	err = imaging.Encode(f, s.image, imaging.JPEG)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func NewImageService(ctx context.Context, url string, zones ...IAZone) ImageAnalyzer {
	return &ImageService{url: url, zones: zones}
}

type Number constraints.Float

// Median calculates the median of a slice of any numeric type
func Median[T Number](data []T) float64 {
	if len(data) == 0 {
		return 0 // Handle empty slice as needed (e.g., return an error or NaN)
	}

	// Create a copy to avoid modifying the original slice
	dataCopy := slices.Clone(data)
	slices.Sort(dataCopy)

	n := len(dataCopy)

	m := float64(n) * float64(0.7)
	// if n%2 == 1 {
	// 	// Odd number of elements
	// 	return float64(dataCopy[n/2])
	// } else {
	// 	// Even number of elements
	// 	mid1 := dataCopy[n/2-1]
	// 	mid2 := dataCopy[n/2]
	// 	return float64(mid1+mid2) / 2.0
	// }
	return float64(dataCopy[int(m)])
}
