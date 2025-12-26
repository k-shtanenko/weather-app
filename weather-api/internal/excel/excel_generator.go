package excel

import (
	"context"
	"fmt"
	"time"

	"github.com/k-shtanenko/weather-app/weather-api/internal/logger"
	"github.com/k-shtanenko/weather-app/weather-api/internal/models"
	"github.com/xuri/excelize/v2"
)

type ExcelGenerator interface {
	GenerateWeatherReport(
		ctx context.Context,
		reportType models.ReportType,
		cityID int,
		cityName string,
		periodStart, periodEnd time.Time,
		weatherData []models.WeatherDataEntity,
		aggregates []models.DailyAggregateEntity,
	) ([]byte, error)
}

type ExcelGeneratorImpl struct {
	logger logger.Logger
}

func NewExcelGeneratorImpl() *ExcelGeneratorImpl {
	return &ExcelGeneratorImpl{
		logger: logger.New("info", "development").WithField("component", "excel_generator"),
	}
}

func (e *ExcelGeneratorImpl) GenerateWeatherReport(
	ctx context.Context,
	reportType models.ReportType,
	cityID int,
	cityName string,
	periodStart, periodEnd time.Time,
	weatherData []models.WeatherDataEntity,
	aggregates []models.DailyAggregateEntity,
) ([]byte, error) {
	e.logger.Infof("Generating %s report for city %s (ID: %v)", reportType, cityName, cityID)

	f := excelize.NewFile()
	defer f.Close()

	f.SetDocProps(&excelize.DocProperties{
		Title:       fmt.Sprintf("Weather Report - %s", reportType),
		Subject:     "Weather Data Analysis",
		Creator:     "Weather API Service",
		Description: fmt.Sprintf("Weather data report for %s, period %s to %s", cityName, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02")),
		Created:     time.Now().String(),
	})

	if err := e.createMainSheet(f, weatherData, reportType, cityName, periodStart, periodEnd); err != nil {
		return nil, fmt.Errorf("failed to create main sheet: %w", err)
	}

	if len(aggregates) > 0 {
		if err := e.createAggregateSheet(f, aggregates); err != nil {
			return nil, fmt.Errorf("failed to create aggregate sheet: %w", err)
		}
	}

	if err := e.createStatisticsSheet(f, weatherData); err != nil {
		return nil, fmt.Errorf("failed to create statistics sheet: %w", err)
	}

	f.DeleteSheet("Sheet1")

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write excel to buffer: %w", err)
	}

	e.logger.Infof("Generated %s report with %d data points", reportType, len(weatherData))
	return buf.Bytes(), nil
}

func (e *ExcelGeneratorImpl) createMainSheet(
	f *excelize.File,
	data []models.WeatherDataEntity,
	reportType models.ReportType,
	cityName string,
	periodStart, periodEnd time.Time,
) error {
	sheetName := "Weather Data"
	f.NewSheet(sheetName)

	headers := []string{
		"Дата и время", "Температура (°C)", "Ощущается как (°C)", "Влажность (%)",
		"Давление (hPa)", "Скорость ветра (м/с)", "Направление ветра", "Облачность (%)",
		"Погода", "Видимость (м)",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	for rowIdx, item := range data {
		row := rowIdx + 2
		colIdx := 1

		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetRecordedAt().Format("2006-01-02 15:04"))
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetTemperature())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetFeelsLike())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetHumidity())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetPressure())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetWindSpeed())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), fmt.Sprintf("%d°", item.GetWindDeg()))
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetClouds())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetWeatherDescription())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), item.GetVisibility())
	}

	titleCell := "A1"
	lastColumn := e.colLetter(len(headers))
	f.SetCellValue(sheetName, titleCell, fmt.Sprintf("Weather Report: %s", cityName))
	f.MergeCell(sheetName, titleCell, lastColumn+"1")

	periodCell := "A2"
	f.SetCellValue(sheetName, periodCell, fmt.Sprintf("Period: %s to %s",
		periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02")))
	f.MergeCell(sheetName, periodCell, lastColumn+"2")

	reportTypeCell := "A3"
	f.SetCellValue(sheetName, reportTypeCell, fmt.Sprintf("Report Type: %s", reportType))
	f.MergeCell(sheetName, reportTypeCell, lastColumn+"3")

	for i := 1; i <= len(headers); i++ {
		colWidth := 15.0
		if i == 1 {
			colWidth = 18.0
		} else if i == 9 {
			colWidth = 20.0
		}
		f.SetColWidth(sheetName, e.colLetter(i), e.colLetter(i), colWidth)
	}

	return nil
}

func (e *ExcelGeneratorImpl) createAggregateSheet(
	f *excelize.File,
	aggregates []models.DailyAggregateEntity,
) error {
	sheetName := "Daily Aggregates"
	f.NewSheet(sheetName)

	headers := []string{
		"Дата", "Средняя температура", "Макс. температура", "Мин. температура",
		"Средняя влажность", "Среднее давление", "Средняя скорость ветра",
		"Преобладающая погода", "Всего записей",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	for rowIdx, agg := range aggregates {
		row := rowIdx + 2
		colIdx := 1

		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetDate().Format("2006-01-02"))
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetAvgTemperature())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetMaxTemperature())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetMinTemperature())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetAvgHumidity())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetAvgPressure())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetAvgWindSpeed())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetDominantWeather())
		colIdx++
		f.SetCellValue(sheetName, e.cell(colIdx, row), agg.GetTotalRecords())
	}

	for i := 1; i <= len(headers); i++ {
		f.SetColWidth(sheetName, e.colLetter(i), e.colLetter(i), 18)
	}

	return nil
}

func (e *ExcelGeneratorImpl) createStatisticsSheet(
	f *excelize.File,
	data []models.WeatherDataEntity,
) error {
	if len(data) == 0 {
		return nil
	}

	sheetName := "Statistics"
	f.NewSheet(sheetName)

	var totalTemp, totalHumidity, totalPressure, totalWindSpeed float64
	minTemp := 1000.0
	maxTemp := -1000.0
	weatherCount := make(map[string]int)

	for _, item := range data {
		temp := item.GetTemperature()
		totalTemp += temp

		if temp < minTemp {
			minTemp = temp
		}
		if temp > maxTemp {
			maxTemp = temp
		}

		totalHumidity += float64(item.GetHumidity())
		totalPressure += float64(item.GetPressure())
		totalWindSpeed += item.GetWindSpeed()

		weatherDesc := item.GetWeatherDescription()
		weatherCount[weatherDesc]++
	}

	avgTemp := totalTemp / float64(len(data))
	avgHumidity := totalHumidity / float64(len(data))
	avgPressure := totalPressure / float64(len(data))
	avgWindSpeed := totalWindSpeed / float64(len(data))

	dominantWeather := ""
	maxCount := 0
	for weather, count := range weatherCount {
		if count > maxCount {
			maxCount = count
			dominantWeather = weather
		}
	}

	stats := []struct {
		label string
		value interface{}
	}{
		{"Total Records", len(data)},
		{"Average Temperature", fmt.Sprintf("%.1f °C", avgTemp)},
		{"Min Temperature", fmt.Sprintf("%.1f °C", minTemp)},
		{"Max Temperature", fmt.Sprintf("%.1f °C", maxTemp)},
		{"Average Humidity", fmt.Sprintf("%.1f %%", avgHumidity)},
		{"Average Pressure", fmt.Sprintf("%.1f hPa", avgPressure)},
		{"Average Wind Speed", fmt.Sprintf("%.2f m/s", avgWindSpeed)},
		{"Dominant Weather", dominantWeather},
	}

	for i, stat := range stats {
		row := i + 1
		f.SetCellValue(sheetName, e.cell(1, row), stat.label)
		f.SetCellValue(sheetName, e.cell(2, row), stat.value)
	}

	f.SetColWidth(sheetName, "A", "A", 25)
	f.SetColWidth(sheetName, "B", "B", 20)

	return nil
}

func (e *ExcelGeneratorImpl) cell(col, row int) string {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	return cell
}

func (e *ExcelGeneratorImpl) colLetter(col int) string {
	letter, _ := excelize.ColumnNumberToName(col)
	return letter
}
