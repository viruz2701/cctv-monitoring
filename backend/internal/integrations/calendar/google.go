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

// ── Google Calendar API ───────────────────────────────────────────────

// GoogleCalendar — адаптер Google Calendar API v3.
//
// API Reference: https://developers.google.com/calendar/api/v3/reference/events
//
// Compliance:
//   - ISO 27001 A.13.1 (Network security — TLS 1.3 via OAuth2 client)
//   - IEC 62443-3-3 SL-2 (DMZ — внешний API через OAuth2)
//   - GDPR Art. 28 (Data Processor — Google Workspace DPA)
type GoogleCalendar struct {
	client *http.Client // OAuth2 HTTP client
	config Config
}

// NewGoogleCalendar создаёт новый Google Calendar адаптер.
func NewGoogleCalendar(client *http.Client, config Config) *GoogleCalendar {
	return &GoogleCalendar{
		client: client,
		config: config,
	}
}

// calendarEvent — внутренняя структура Google Calendar Event.
type googleEvent struct {
	ID          string           `json:"id,omitempty"`
	Summary     string           `json:"summary"`
	Description string           `json:"description"`
	Start       *googleEventTime `json:"start"`
	End         *googleEventTime `json:"end"`
	Location    string           `json:"location,omitempty"`
	Status      string           `json:"status,omitempty"` // confirmed, tentative, cancelled
	Attendees   []googleAttendee `json:"attendees,omitempty"`
	Source      *googleSource    `json:"source,omitempty"`
}

type googleEventTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone,omitempty"`
}

type googleAttendee struct {
	Email          string `json:"email"`
	DisplayName    string `json:"displayName,omitempty"`
	ResponseStatus string `json:"responseStatus,omitempty"`
}

type googleSource struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// googleCalendarID возвращает ID календаря (primary или указанный).
const googleCalendarScope = "primary"

// ── CreateEvent ───────────────────────────────────────────────────────

// CreateEvent создаёт событие в Google Calendar.
// Возвращает Google event ID.
//
// Endpoint: POST /calendars/{calendarId}/events
func (gc *GoogleCalendar) CreateEvent(ctx context.Context, wo WorkOrderEvent) (string, error) {
	event := gc.buildEvent(wo)
	event.Status = "confirmed"

	body, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("google marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		gc.eventURL(""), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("google create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := gc.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("google create event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", gc.parseError(resp)
	}

	var created googleEvent
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("google decode response: %w", err)
	}

	return created.ID, nil
}

// ── UpdateEvent ───────────────────────────────────────────────────────

// UpdateEvent обновляет событие в Google Calendar.
//
// Endpoint: PUT /calendars/{calendarId}/events/{eventId}
func (gc *GoogleCalendar) UpdateEvent(ctx context.Context, eventID string, wo WorkOrderEvent) error {
	event := gc.buildEvent(wo)
	event.Status = "confirmed"

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("google marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		gc.eventURL(eventID), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("google update request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := gc.client.Do(req)
	if err != nil {
		return fmt.Errorf("google update event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return gc.parseError(resp)
	}

	return nil
}

// ── DeleteEvent ───────────────────────────────────────────────────────

// DeleteEvent удаляет событие из Google Calendar.
//
// Endpoint: DELETE /calendars/{calendarId}/events/{eventId}
func (gc *GoogleCalendar) DeleteEvent(ctx context.Context, eventID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		gc.eventURL(eventID), nil)
	if err != nil {
		return fmt.Errorf("google delete request: %w", err)
	}

	resp, err := gc.client.Do(req)
	if err != nil {
		return fmt.Errorf("google delete event: %w", err)
	}
	defer resp.Body.Close()

	// Google returns 204 on successful deletion
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return gc.parseError(resp)
	}

	return nil
}

// ── SyncChanges ───────────────────────────────────────────────────────

// SyncChanges получает изменения из Google Calendar с момента since.
//
// Endpoint: GET /calendars/{calendarId}/events
// Parameters: timeMin, showDeleted=true, orderBy=updated
func (gc *GoogleCalendar) SyncChanges(ctx context.Context, since time.Time) ([]CalendarChange, error) {
	url := fmt.Sprintf("%s?timeMin=%s&showDeleted=true&orderBy=updated&singleEvents=true",
		gc.eventURL(""),
		since.Format(time.RFC3339))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("google sync request: %w", err)
	}

	resp, err := gc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google sync changes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, gc.parseError(resp)
	}

	var list struct {
		Items []googleEvent `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("google decode sync: %w", err)
	}

	changes := make([]CalendarChange, 0, len(list.Items))
	for _, item := range list.Items {
		changeType := "updated"
		if item.Status == "cancelled" {
			changeType = "deleted"
		}

		// Определяем время изменения из updated (или created)
		changedAt := time.Now()
		if item.Start != nil && item.Start.DateTime != "" {
			if t, err := time.Parse(time.RFC3339, item.Start.DateTime); err == nil {
				changedAt = t
			}
		}

		changes = append(changes, CalendarChange{
			EventID:    item.ID,
			Type:       changeType,
			ExternalID: item.ID,
			Provider:   "google",
			ChangedAt:  changedAt,
		})
	}

	return changes, nil
}

// ── Helpers ───────────────────────────────────────────────────────────

// eventURL возвращает URL для Google Calendar Events API.
func (gc *GoogleCalendar) eventURL(eventID string) string {
	if eventID == "" {
		return fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events", googleCalendarScope)
	}
	return fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events/%s",
		googleCalendarScope, eventID)
}

// buildEvent конвертирует WorkOrderEvent в Google Calendar event.
func (gc *GoogleCalendar) buildEvent(wo WorkOrderEvent) googleEvent {
	desc := wo.Description
	if wo.AssignedTo != "" {
		desc = fmt.Sprintf("%s\n\nAssigned to: %s", desc, wo.AssignedTo)
	}

	ge := googleEvent{
		Summary:     wo.Title,
		Description: desc,
		Start: &googleEventTime{
			DateTime: wo.StartTime.Format(time.RFC3339),
			TimeZone: "UTC",
		},
		End: &googleEventTime{
			DateTime: wo.EndTime.Format(time.RFC3339),
			TimeZone: "UTC",
		},
		Location: wo.Location,
		Source: &googleSource{
			Title: "CCTV Health Monitor",
			URL:   fmt.Sprintf("https://cms.internal/work-orders/%s", wo.ID),
		},
	}

	// Если WO cancelled — создаём cancelled событие
	if wo.Status == "cancelled" {
		ge.Status = "cancelled"
	}

	return ge
}

// parseError разбирает ошибку Google Calendar API.
func (gc *GoogleCalendar) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &apiErr); err != nil {
		return fmt.Errorf("google API %d: %s", resp.StatusCode, string(body))
	}
	return fmt.Errorf("google API %d: %s", apiErr.Error.Code, apiErr.Error.Message)
}

// compile-time interface check
var _ CalendarProvider = (*GoogleCalendar)(nil)
