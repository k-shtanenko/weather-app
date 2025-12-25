package entities

import (
	"time"
)

type ReportType string

const (
	ReportTypeDaily   ReportType = "daily"
	ReportTypeWeekly  ReportType = "weekly"
	ReportTypeMonthly ReportType = "monthly"
)

type ExcelReportEntity interface {
	GetID() string
	GetReportType() ReportType
	GetCityID() int
	GetPeriodStart() time.Time
	GetPeriodEnd() time.Time
	GetFileName() string
	GetFileSize() int64
	GetStoragePath() string
	GetDownloadURL() string
	GetChecksum() string
	GetGeneratedAt() time.Time
	GetExpiresAt() *time.Time
}

type ExcelReport struct {
	ID          string     `json:"id" db:"id"`
	ReportType  ReportType `json:"report_type" db:"report_type"`
	CityID      int        `json:"city_id,omitempty" db:"city_id"`
	PeriodStart time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd   time.Time  `json:"period_end" db:"period_end"`
	FileName    string     `json:"file_name" db:"file_name"`
	FileSize    int64      `json:"file_size" db:"file_size"`
	StoragePath string     `json:"storage_path" db:"storage_path"`
	DownloadURL string     `json:"download_url,omitempty" db:"download_url"`
	Checksum    string     `json:"checksum" db:"checksum"`
	GeneratedAt time.Time  `json:"generated_at" db:"generated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

func (e *ExcelReport) GetID() string             { return e.ID }
func (e *ExcelReport) GetReportType() ReportType { return e.ReportType }
func (e *ExcelReport) GetCityID() int            { return e.CityID }
func (e *ExcelReport) GetPeriodStart() time.Time { return e.PeriodStart }
func (e *ExcelReport) GetPeriodEnd() time.Time   { return e.PeriodEnd }
func (e *ExcelReport) GetFileName() string       { return e.FileName }
func (e *ExcelReport) GetFileSize() int64        { return e.FileSize }
func (e *ExcelReport) GetStoragePath() string    { return e.StoragePath }
func (e *ExcelReport) GetDownloadURL() string    { return e.DownloadURL }
func (e *ExcelReport) GetChecksum() string       { return e.Checksum }
func (e *ExcelReport) GetGeneratedAt() time.Time { return e.GeneratedAt }
func (e *ExcelReport) GetExpiresAt() *time.Time  { return e.ExpiresAt }
