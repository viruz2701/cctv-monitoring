package calendar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ── Outlook Calendar API (Microsoft Graph) ────────────────────────────

// OutlookCalendar — адаптер Microsoft Graph Calendar API.
//
// API Reference: https://learn.microsoft.com/en-us/graph/api/resources/event
//
// Compliance:
//   - ISO 27001 A.13.1 (Network security — TLS 1.3 via OAuth2 client)
//   - IEC 62443-3-3 SL-2 (DMZ — внешний API через OAuth2)
//   - GDPR Art. 28 (Data Processor — Microsoft 365 DPA)
type OutlookCalendar struct {
	client *http.Client // OAuth2 HTTP client
	config Config
}

// NewOutlookCalendar создаёт новый Outlook Calendar адаптер.
func NewOutlookCalendar(client *http.Client, config Config) *OutlookCalendar {
	return &OutlookCalendar{
		client: client,
		config: config,
	}
}

// outlookEvent — внутренняя структура Microsoft Graph Event.
type outlookEvent struct {
	ID             string                 `json:"id,omitempty"`
	Subject        string                 `json:"subject"`
	Body           *outlookItemBody       `json:"body,omitempty"`
	Start          *outlookDateTimeTZ     `json:"start"`
	End            *outlookDateTimeTZ     `json:"end"`
	Location       *outlookLocation       `json:"location,omitempty"`
	Attendees      []outlookAttendee      `json:"attendees,omitempty"`
	IsCancelled    bool                   `json:"isCancelled,omitempty"`
	ShowAs         string                 `json:"showAs,omitempty"` // free, tentative, busy, oof
	ResponseStatus *outlookResponseStatus `json:"responseStatus,omitempty"`
	WebLink        string                 `json:"webLink,omitempty"`
	Categories     []string               `json:"categories,omitempty"`
}

type outlookItemBody struct {
	ContentType string `json:"contentType"` // text, html
	Content     string `json:"content"`
}

type outlookDateTimeTZ struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

type outlookLocation struct {
	DisplayName string `json:"displayName"`
}

type outlookAttendee struct {
	EmailAddress *outlookEmailAddress `json:"emailAddress"`
	Type         string               `json:"type"` // required, optional
}

type outlookEmailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type outlookResponseStatus struct {
	Response string `json:"response"`
	Time     string `json:"time"`
}

// ── CreateEvent ───────────────────────────────────────────────────────

// CreateEvent создаёт событие в Outlook Calendar.
// Возвращает Outlook event ID.
//
// Endpoint: POST /me/events
func (oc *OutlookCalendar) CreateEvent(ctx context.Context, wo WorkOrderEvent) (string, error) {
	event := oc.buildEvent(wo)
	if wo.Status == "cancelled" {
		event.IsCancelled = true
	}

	body, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("outlook marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://graph.microsoft.com/v1.0/me/events", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("outlook create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := oc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("outlook create event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", oc.parseError(resp)
	}

	var created outlookEvent
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("outlook decode response: %w", err)
	}

	return created.ID, nil
}

// ── UpdateEvent ───────────────────────────────────────────────────────

// UpdateEvent обновляет событие в Outlook Calendar.
//
// Endpoint: PATCH /me/events/{eventId}
func (oc *OutlookCalendar) UpdateEvent(ctx context.Context, eventID string, wo WorkOrderEvent) error {
	event := oc.buildEvent(wo)
	if wo.Status == "cancelled" {
		event.IsCancelled = true
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("outlook marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		fmt.Sprintf("https://graph.microsoft.com/v1.0/me/events/%s", eventID),
		bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("outlook update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := oc.client.Do(req)
	if err != nil {
		return fmt.Errorf("outlook update event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return oc.parseError(resp)
	}

	return nil
}

// ── DeleteEvent ───────────────────────────────────────────────────────

// DeleteEvent удаляет событие из Outlook Calendar.
//
// Endpoint: DELETE /me/events/{eventId}
func (oc *OutlookCalendar) DeleteEvent(ctx context.Context, eventID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("https://graph.microsoft.com/v1.0/me/events/%s", eventID), nil)
	if err != nil {
		return fmt.Errorf("outlook delete request: %w", err)
	}

	resp, err := oc.client.Do(req)
	if err != nil {
		return fmt.Errorf("outlook delete event: %w", err)
	}
	defer resp.Body.Close()

	// Graph API returns 204 on successful deletion
	if resp.StatusCode != http.StatusNoContent {
		return oc.parseError(resp)
	}

	return nil
}

// ── SyncChanges ───────────────────────────────────────────────────────

// SyncChanges получает изменения из Outlook Calendar с момента since.
//
// Endpoint: GET /me/calendarView?startDateTime=...&endDateTime=...
func (oc *OutlookCalendar) SyncChanges(ctx context.Context, since time.Time) ([]CalendarChange, error) {
	// Outlook calendarView требует start/endDateTime
	until := since.Add(oc.config.SyncWindow)

	url := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/me/calendarView?"+
			"startDateTime=%s&endDateTime=%s&$orderBy=start/dateTime&$top=100",
		since.Format(time.RFC3339),
		until.Format(time.RFC3339),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("outlook sync request: %w", err)
	}
	req.Header.Set("Prefer", "outlook.timezone=\"UTC\"")

	resp, err := oc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("outlook sync changes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, oc.parseError(resp)
	}

	var list struct {
		Value []outlookEvent `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("outlook decode sync: %w", err)
	}

	changes := make([]CalendarChange, 0, len(list.Value))
	for _, item := range list.Value {
		changeType := "updated"
		if item.IsCancelled {
			changeType = "deleted"
		}

		changedAt := time.Now()
		if item.Start != nil && item.Start.DateTime != "" {
			if t, err := time.Parse("2006-01-02T15:04:05", item.Start.DateTime); err == nil {
				changedAt = t
			}
		}

		changes = append(changes, CalendarChange{
			EventID:    item.ID,
			Type:       changeType,
			ExternalID: item.ID,
			Provider:   "outlook",
			ChangedAt:  changedAt,
		})
	}

	return changes, nil
}

// ── Helpers ───────────────────────────────────────────────────────────

// buildEvent конвертирует WorkOrderEvent в Outlook Calendar event.
func (oc *OutlookCalendar) buildEvent(wo WorkOrderEvent) outlookEvent {
	contentType := "text"
	content := wo.Description
	if wo.AssignedTo != "" {
		content = fmt.Sprintf("%s\n\nAssigned to: %s", content, wo.AssignedTo)
	}

	oe := outlookEvent{
		Subject: wo.Title,
		Body: &outlookItemBody{
			ContentType: contentType,
			Content:     content,
		},
		Start: &outlookDateTimeTZ{
			DateTime: wo.StartTime.Format("2006-01-02T15:04:05"),
			TimeZone: "UTC",
		},
		End: &outlookDateTimeTZ{
			DateTime: wo.EndTime.Format("2006-01-02T15:04:05"),
			TimeZone: "UTC",
		},
		ShowAs:     "busy",
		Categories: []string{"CCTV", "WorkOrder"},
	}

	if wo.Location != "" {
		oe.Location = &outlookLocation{DisplayName: wo.Location}
	}

	return oe
}

// parseError разбирает ошибку Microsoft Graph API.
func (oc *OutlookCalendar) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("outlook API %d: %s", resp.StatusCode, string(body))
	}
	return fmt.Errorf("outlook API %s: %s", apiErr.Error.Code, apiErr.Error.Message)
}

// compile-time interface check
var _ CalendarProvider = (*OutlookCalendar)(nil)
