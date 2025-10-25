package util

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fogleman/gg"
	"github.com/justinndidit/forex/internal/model"
	"github.com/rs/zerolog"
	"golang.org/x/image/font/basicfont"
)

const (
	imageWidth  = 1000
	imageHeight = 700
	padding     = 50
	summaryPath = "cache/summary.png"
)

type ImageService struct {
	log *zerolog.Logger
}

func NewImageService(log *zerolog.Logger) *ImageService {
	return &ImageService{log: log}
}

func (s *ImageService) GenerateSummary(totalCountries int, topCountries []model.CountryDBRow, refreshTime time.Time) error {
	// Create drawing context
	dc := gg.NewContext(imageWidth, imageHeight)

	// Draw gradient background
	s.drawGradientBackground(dc)

	// Draw main content box with shadow
	boxX := float64(padding)
	boxY := float64(padding)
	boxWidth := float64(imageWidth - 2*padding)
	boxHeight := float64(imageHeight - 2*padding)

	// Shadow
	dc.SetRGBA(0, 0, 0, 0.2)
	dc.DrawRoundedRectangle(boxX+5, boxY+5, boxWidth, boxHeight, 15)
	dc.Fill()

	// White box
	dc.SetRGB(1, 1, 1)
	dc.DrawRoundedRectangle(boxX, boxY, boxWidth, boxHeight, 15)
	dc.Fill()

	// Draw border
	dc.SetRGB(0.8, 0.8, 0.8)
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(boxX, boxY, boxWidth, boxHeight, 15)
	dc.Stroke()

	// Draw header section with colored background
	dc.SetRGB(0.2, 0.4, 0.7) // Blue header
	dc.DrawRoundedRectangle(boxX, boxY, boxWidth, 100, 15)
	dc.Fill()

	// Draw title using basic font (no external font needed!)
	dc.SetFontFace(basicfont.Face7x13)
	dc.SetRGB(1, 1, 1) // White text
	title := "COUNTRY GDP SUMMARY REPORT"
	titleWidth := float64(len(title) * 7) // 7 pixels per character
	dc.DrawString(title, (imageWidth-titleWidth)/2, boxY+45)

	// Draw metadata section
	currentY := boxY + 130
	dc.SetRGB(0.3, 0.3, 0.3) // Dark gray text

	// Last refreshed
	refreshStr := fmt.Sprintf("Last Refreshed: %s", refreshTime.Format("Mon, 02 Jan 2006 15:04:05 MST"))
	dc.DrawString(refreshStr, boxX+30, currentY)
	currentY += 30

	// Total countries
	totalStr := fmt.Sprintf("Total Countries Analyzed: %d", totalCountries)
	dc.DrawString(totalStr, boxX+30, currentY)
	currentY += 45

	// Draw divider line
	dc.SetRGB(0.85, 0.85, 0.85)
	dc.SetLineWidth(2)
	dc.DrawLine(boxX+30, currentY, imageWidth-boxX-30, currentY)
	dc.Stroke()
	currentY += 35

	// Draw "Top 5" header
	dc.SetRGB(0.2, 0.4, 0.7)
	dc.DrawString("TOP 5 COUNTRIES BY ESTIMATED GDP", boxX+30, currentY)
	currentY += 45

	// Draw top 5 countries list
	lineHeight := 45.0
	for i, country := range topCountries {
		y := currentY + (float64(i) * lineHeight)

		// Alternating row background
		if i%2 == 0 {
			dc.SetRGBA(0.95, 0.97, 1.0, 1.0) // Light blue
			dc.DrawRoundedRectangle(boxX+20, y-20, boxWidth-40, lineHeight-5, 5)
			dc.Fill()
		}

		// Rank number with circle
		dc.SetRGB(0.2, 0.4, 0.7)
		dc.DrawCircle(boxX+50, y, 18)
		dc.Fill()

		// Rank number text
		dc.SetRGB(1, 1, 1)
		rankStr := fmt.Sprintf("%d", i+1)
		rankX := boxX + 50 - float64(len(rankStr)*7)/2
		dc.DrawString(rankStr, rankX, y-3)

		// Country name
		dc.SetRGB(0.2, 0.2, 0.2)
		dc.DrawString(country.Name, boxX+90, y)

		// GDP value (right-aligned)
		var gdpStr string
		if country.EstimatedGDP.Valid {
			// Format with commas
			gdpStr = s.formatCurrency(country.EstimatedGDP.Float64)
		} else {
			gdpStr = "N/A"
		}

		dc.SetRGB(0.1, 0.6, 0.3) // Green for money
		gdpWidth := float64(len(gdpStr) * 7)
		dc.DrawString(gdpStr, imageWidth-boxX-50-gdpWidth, y)
	}

	// Draw footer
	footerY := float64(imageHeight - padding - 15)
	dc.SetRGB(0.6, 0.6, 0.6)
	footerText := fmt.Sprintf("Generated on %s", time.Now().Format("2006-01-02 15:04:05"))
	footerWidth := float64(len(footerText) * 7)
	dc.DrawString(footerText, (imageWidth-footerWidth)/2, footerY)

	// Save image
	cacheDir := filepath.Dir(summaryPath)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		s.log.Error().Err(err).Str("path", cacheDir).Msg("Failed to create cache directory")
		return err
	}

	if err := dc.SavePNG(summaryPath); err != nil {
		s.log.Error().Err(err).Str("path", summaryPath).Msg("Failed to save summary image")
		return err
	}

	s.log.Info().Str("path", summaryPath).Msg("Summary image generated successfully")
	return nil
}

// drawGradientBackground creates a gradient background
func (s *ImageService) drawGradientBackground(dc *gg.Context) {
	for y := 0; y < imageHeight; y++ {
		ratio := float64(y) / float64(imageHeight)
		r := 0.9 + (ratio * 0.1)
		g := 0.95 + (ratio * 0.05)
		b := 1.0
		dc.SetRGB(r, g, b)
		dc.DrawLine(0, float64(y), float64(imageWidth), float64(y))
		dc.Stroke()
	}
}

// formatCurrency formats a number with commas as thousands separator
func (s *ImageService) formatCurrency(value float64) string {
	// Simple currency formatting with commas
	str := fmt.Sprintf("%.2f", value)

	// Add commas
	parts := []rune(str)
	dotIndex := len(str) - 3 // Position of decimal point

	result := string(parts[dotIndex:]) // Decimal part
	intPart := parts[:dotIndex]

	// Add commas to integer part
	for i := len(intPart) - 1; i >= 0; i-- {
		result = string(intPart[i]) + result
		if (len(intPart)-i)%3 == 0 && i != 0 {
			result = "," + result
		}
	}

	return "$" + result
}
