// Package compliance — In-memory stores for PersonalDataStore and GDPRStore.
//
// Используется для MVP/development. В production заменить на PostgreSQL.
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — логирование всех операций)
//   - OWASP ASVS V8 (Data Protection — данные в памяти шифруются через crypto providers)
package compliance

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// MemoryPersonalDataStore — in-memory реализация PersonalDataStore
// ═══════════════════════════════════════════════════════════════════════

// MemoryPersonalDataStore implements PersonalDataStore using in-memory maps.
type MemoryPersonalDataStore struct {
	mu        sync.RWMutex
	consents  map[string]*ConsentRecord
	dsars     map[string]*DSARRequest
	inventory map[string]*DataInventoryItem
	logger    *slog.Logger
}

// NewMemoryPersonalDataStore creates a new in-memory personal data store.
func NewMemoryPersonalDataStore(logger *slog.Logger) *MemoryPersonalDataStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &MemoryPersonalDataStore{
		consents:  make(map[string]*ConsentRecord),
		dsars:     make(map[string]*DSARRequest),
		inventory: make(map[string]*DataInventoryItem),
		logger:    logger.With("component", "compliance.memory_store"),
	}
}

func (s *MemoryPersonalDataStore) SaveConsent(ctx interface{}, record *ConsentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consents[record.ID] = record
	return nil
}

func (s *MemoryPersonalDataStore) GetConsent(ctx interface{}, id string) (*ConsentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.consents[id]
	if !ok {
		return nil, nil
	}
	return record, nil
}

func (s *MemoryPersonalDataStore) ListConsents(ctx interface{}, subjectID string) ([]*ConsentRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*ConsentRecord
	for _, record := range s.consents {
		if record.SubjectID == subjectID {
			result = append(result, record)
		}
	}
	return result, nil
}

func (s *MemoryPersonalDataStore) RevokeConsent(ctx interface{}, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.consents[id]
	if !ok {
		return fmt.Errorf("consent not found: %s", id)
	}
	now := time.Now().UTC()
	record.Status = ConsentStatusRevoked
	record.RevokedAt = &now
	record.UpdatedAt = now
	return nil
}

func (s *MemoryPersonalDataStore) SaveDSAR(ctx interface{}, request *DSARRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dsars[request.ID] = request
	return nil
}

func (s *MemoryPersonalDataStore) GetDSAR(ctx interface{}, id string) (*DSARRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.dsars[id]
	if !ok {
		return nil, nil
	}
	return req, nil
}

func (s *MemoryPersonalDataStore) ListDSARs(ctx interface{}, subjectID string) ([]*DSARRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*DSARRequest
	for _, req := range s.dsars {
		if req.SubjectID == subjectID {
			result = append(result, req)
		}
	}
	return result, nil
}

func (s *MemoryPersonalDataStore) UpdateDSARStatus(ctx interface{}, id string, status DSARStatus, responseData, rejectionReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.dsars[id]
	if !ok {
		return fmt.Errorf("DSAR not found: %s", id)
	}
	now := time.Now().UTC()
	req.Status = status
	req.ResponseData = responseData
	req.RejectionReason = rejectionReason
	if status == DSARStatusFulfilled {
		req.FulfilledAt = &now
	}
	req.UpdatedAt = now
	return nil
}

func (s *MemoryPersonalDataStore) SaveInventoryItem(ctx interface{}, item *DataInventoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inventory[item.ID] = item
	return nil
}

func (s *MemoryPersonalDataStore) ListInventory(ctx interface{}) ([]*DataInventoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DataInventoryItem, 0, len(s.inventory))
	for _, item := range s.inventory {
		result = append(result, item)
	}
	return result, nil
}

func (s *MemoryPersonalDataStore) GetInventoryItem(ctx interface{}, id string) (*DataInventoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.inventory[id]
	if !ok {
		return nil, nil
	}
	return item, nil
}

// ═══════════════════════════════════════════════════════════════════════
// MemoryGDPRStore — in-memory реализация GDPRStore
// ═══════════════════════════════════════════════════════════════════════

// MemoryGDPRStore implements GDPRStore using in-memory maps.
type MemoryGDPRStore struct {
	mu                 sync.RWMutex
	erasures           map[string]*ErasureRequest
	portabilityExports map[string]*PortabilityExport
	consentAudit       map[string]*ConsentAuditEntry
	dpias              map[string]*DPIAReport
	transfers          map[string]*DataTransferAgreement
	logger             *slog.Logger
}

// NewMemoryGDPRStore creates a new in-memory GDPR store.
func NewMemoryGDPRStore(logger *slog.Logger) *MemoryGDPRStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &MemoryGDPRStore{
		erasures:           make(map[string]*ErasureRequest),
		portabilityExports: make(map[string]*PortabilityExport),
		consentAudit:       make(map[string]*ConsentAuditEntry),
		dpias:              make(map[string]*DPIAReport),
		transfers:          make(map[string]*DataTransferAgreement),
		logger:             logger.With("component", "compliance.memory_gdpr_store"),
	}
}

// ── Erasure ─────────────────────────────────────────────────────────

func (s *MemoryGDPRStore) SaveErasureRequest(ctx interface{}, req *ErasureRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.erasures[req.ID] = req
	return nil
}

func (s *MemoryGDPRStore) GetErasureRequest(ctx interface{}, id string) (*ErasureRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	req, ok := s.erasures[id]
	if !ok {
		return nil, nil
	}
	return req, nil
}

func (s *MemoryGDPRStore) ListErasureRequests(ctx interface{}, subjectID string) ([]*ErasureRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*ErasureRequest
	for _, req := range s.erasures {
		if req.SubjectID == subjectID {
			result = append(result, req)
		}
	}
	return result, nil
}

func (s *MemoryGDPRStore) UpdateErasureStatus(ctx interface{}, id string, status ErasureRequestStatus, rejectionReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, ok := s.erasures[id]
	if !ok {
		return fmt.Errorf("erasure request not found: %s", id)
	}
	now := time.Now().UTC()
	req.Status = status
	req.RejectionReason = rejectionReason
	if status == ErasureStatusCompleted {
		req.CompletedAt = &now
	}
	req.UpdatedAt = now
	return nil
}

// ── Portability ────────────────────────────────────────────────────

func (s *MemoryGDPRStore) SavePortabilityExport(ctx interface{}, exp *PortabilityExport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.portabilityExports[exp.ID] = exp
	return nil
}

func (s *MemoryGDPRStore) GetPortabilityExport(ctx interface{}, id string) (*PortabilityExport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exp, ok := s.portabilityExports[id]
	if !ok {
		return nil, nil
	}
	return exp, nil
}

func (s *MemoryGDPRStore) ListPortabilityExports(ctx interface{}, subjectID string) ([]*PortabilityExport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*PortabilityExport
	for _, exp := range s.portabilityExports {
		if exp.SubjectID == subjectID {
			result = append(result, exp)
		}
	}
	return result, nil
}

// ── Consent Audit ──────────────────────────────────────────────────

func (s *MemoryGDPRStore) SaveConsentAuditEntry(ctx interface{}, entry *ConsentAuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.consentAudit[entry.ID] = entry
	return nil
}

func (s *MemoryGDPRStore) ListConsentAuditEntries(ctx interface{}, subjectID string) ([]*ConsentAuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*ConsentAuditEntry
	for _, entry := range s.consentAudit {
		if entry.SubjectID == subjectID {
			result = append(result, entry)
		}
	}
	return result, nil
}

// ── DPIA ───────────────────────────────────────────────────────────

func (s *MemoryGDPRStore) SaveDPIAReport(ctx interface{}, report *DPIAReport) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dpias[report.ID] = report
	return nil
}

func (s *MemoryGDPRStore) GetDPIAReport(ctx interface{}, id string) (*DPIAReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.dpias[id]
	if !ok {
		return nil, nil
	}
	return report, nil
}

func (s *MemoryGDPRStore) ListDPIAReports(ctx interface{}) ([]*DPIAReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DPIAReport, 0, len(s.dpias))
	for _, report := range s.dpias {
		result = append(result, report)
	}
	return result, nil
}

// ── Data Transfers ─────────────────────────────────────────────────

func (s *MemoryGDPRStore) SaveTransferAgreement(ctx interface{}, agreement *DataTransferAgreement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transfers[agreement.ID] = agreement
	return nil
}

func (s *MemoryGDPRStore) GetTransferAgreement(ctx interface{}, id string) (*DataTransferAgreement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	agreement, ok := s.transfers[id]
	if !ok {
		return nil, nil
	}
	return agreement, nil
}

func (s *MemoryGDPRStore) ListTransferAgreements(ctx interface{}) ([]*DataTransferAgreement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DataTransferAgreement, 0, len(s.transfers))
	for _, agreement := range s.transfers {
		result = append(result, agreement)
	}
	return result, nil
}
