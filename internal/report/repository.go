package report

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"runtime"
	"seanime/internal/constants"
	"seanime/internal/database/models"
	"seanime/internal/library/anime"
)

type Repository struct {
	logger *zerolog.Logger

	savedIssueReport mo.Option[*IssueReport]
}

func NewRepository(logger *zerolog.Logger) *Repository {
	return &Repository{
		logger:           logger,
		savedIssueReport: mo.None[*IssueReport](),
	}
}

type SaveIssueReportOptions struct {
	LogsDir             string                 `json:"logsDir"`
	UserAgent           string                 `json:"userAgent"`
	ClickLogs           []*ClickLog            `json:"clickLogs"`
	NetworkLogs         []*NetworkLog          `json:"networkLogs"`
	ReactQueryLogs      []*ReactQueryLog       `json:"reactQueryLogs"`
	ConsoleLogs         []*ConsoleLog          `json:"consoleLogs"`
	LocalFiles          []*anime.LocalFile     `json:"localFiles"`
    Settings            *models.Settings       `json:"settings"`
    IsAnimeLibraryIssue bool                   `json:"isAnimeLibraryIssue"`
    ServerStatus        interface{}            `json:"serverStatus"`
}

func (r *Repository) SaveIssueReport(opts SaveIssueReportOptions) error {

	var toRedact []string
	if opts.Settings != nil {
		toRedact = opts.Settings.GetSensitiveValues()
	}
	toRedact = lo.Filter(toRedact, func(s string, _ int) bool {
		return s != ""
	})

	issueReport, err := NewIssueReport(
		opts.UserAgent,
		constants.Version,
		runtime.GOOS,
		runtime.GOARCH,
		opts.LogsDir,
		opts.IsAnimeLibraryIssue,
		opts.ServerStatus,
		toRedact,
	)
	if err != nil {
		return fmt.Errorf("failed to create issue report: %w", err)
	}

	issueReport.ClickLogs = opts.ClickLogs
	issueReport.NetworkLogs = opts.NetworkLogs
	issueReport.ReactQueryLogs = opts.ReactQueryLogs
	issueReport.ConsoleLogs = opts.ConsoleLogs

	r.savedIssueReport = mo.Some(issueReport)

	return nil
}

func (r *Repository) GetSavedIssueReport() (*IssueReport, bool) {
	if r.savedIssueReport.IsAbsent() {
		return nil, false
	}

	return r.savedIssueReport.MustGet(), true
}
