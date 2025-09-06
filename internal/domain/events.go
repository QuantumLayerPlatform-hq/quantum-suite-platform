package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BaseDomainEvent provides common functionality for domain events
type BaseDomainEvent struct {
	eventID       string
	eventType     string
	aggregateID   string
	aggregateType string
	timestamp     time.Time
	version       int64
	metadata      map[string]interface{}
}

func NewBaseDomainEvent(eventType, aggregateID, aggregateType string, version int64) BaseDomainEvent {
	return BaseDomainEvent{
		eventID:       uuid.New().String(),
		eventType:     eventType,
		aggregateID:   aggregateID,
		aggregateType: aggregateType,
		timestamp:     time.Now(),
		version:       version,
		metadata:      make(map[string]interface{}),
	}
}

func (e BaseDomainEvent) EventID() string                    { return e.eventID }
func (e BaseDomainEvent) EventType() string                 { return e.eventType }
func (e BaseDomainEvent) AggregateID() string               { return e.aggregateID }
func (e BaseDomainEvent) AggregateType() string             { return e.aggregateType }
func (e BaseDomainEvent) Timestamp() time.Time              { return e.timestamp }
func (e BaseDomainEvent) Version() int64                    { return e.version }
func (e BaseDomainEvent) Metadata() map[string]interface{}  { return e.metadata }

// Core Domain Events

// Tenant Events
type TenantCreated struct {
	BaseDomainEvent
	TenantID TenantID `json:"tenant_id"`
	Name     string   `json:"name"`
	Plan     string   `json:"plan"`
}

type TenantUpdated struct {
	BaseDomainEvent
	TenantID TenantID               `json:"tenant_id"`
	Changes  map[string]interface{} `json:"changes"`
}

type TenantDeactivated struct {
	BaseDomainEvent
	TenantID TenantID `json:"tenant_id"`
	Reason   string   `json:"reason"`
}

// User Events
type UserRegistered struct {
	BaseDomainEvent
	UserID   UserID   `json:"user_id"`
	TenantID TenantID `json:"tenant_id"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
}

type UserUpdated struct {
	BaseDomainEvent
	UserID  UserID                 `json:"user_id"`
	Changes map[string]interface{} `json:"changes"`
}

type UserDeactivated struct {
	BaseDomainEvent
	UserID UserID `json:"user_id"`
	Reason string `json:"reason"`
}

// Project Events
type ProjectCreated struct {
	BaseDomainEvent
	ProjectID   ProjectID `json:"project_id"`
	TenantID    TenantID  `json:"tenant_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
}

type ProjectUpdated struct {
	BaseDomainEvent
	ProjectID ProjectID              `json:"project_id"`
	Changes   map[string]interface{} `json:"changes"`
}

type ProjectDeleted struct {
	BaseDomainEvent
	ProjectID ProjectID `json:"project_id"`
	Reason    string    `json:"reason"`
}

// Workspace Events
type WorkspaceCreated struct {
	BaseDomainEvent
	WorkspaceID WorkspaceID `json:"workspace_id"`
	ProjectID   ProjectID   `json:"project_id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
}

type WorkspaceUpdated struct {
	BaseDomainEvent
	WorkspaceID WorkspaceID            `json:"workspace_id"`
	Changes     map[string]interface{} `json:"changes"`
}

type WorkspaceDeleted struct {
	BaseDomainEvent
	WorkspaceID WorkspaceID `json:"workspace_id"`
	Reason      string      `json:"reason"`
}

// QAgent Events
type CodeGenerationRequested struct {
	BaseDomainEvent
	GenerationID string      `json:"generation_id"`
	AgentID      AgentID     `json:"agent_id"`
	WorkspaceID  WorkspaceID `json:"workspace_id"`
	Prompt       string      `json:"prompt"`
	Language     string      `json:"language"`
	Context      map[string]interface{} `json:"context"`
}

type CodeGenerated struct {
	BaseDomainEvent
	GenerationID string                 `json:"generation_id"`
	AgentID      AgentID                `json:"agent_id"`
	Code         string                 `json:"code"`
	Language     string                 `json:"language"`
	Validation   map[string]interface{} `json:"validation"`
	TokensUsed   int                    `json:"tokens_used"`
	Cost         float64                `json:"cost"`
}

type CodeGenerationFailed struct {
	BaseDomainEvent
	GenerationID string `json:"generation_id"`
	AgentID      AgentID `json:"agent_id"`
	Error        string `json:"error"`
	Reason       string `json:"reason"`
}

// QTest Events
type TestSuiteCreated struct {
	BaseDomainEvent
	SuiteID     string    `json:"suite_id"`
	ProjectID   ProjectID `json:"project_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	TestCount   int       `json:"test_count"`
}

type TestsGenerated struct {
	BaseDomainEvent
	SuiteID         string                 `json:"suite_id"`
	GeneratedTests  []map[string]interface{} `json:"generated_tests"`
	CoverageTarget  float64                `json:"coverage_target"`
	GenerationTime  time.Duration          `json:"generation_time"`
}

type TestExecutionStarted struct {
	BaseDomainEvent
	ExecutionID string `json:"execution_id"`
	SuiteID     string `json:"suite_id"`
	TestCount   int    `json:"test_count"`
}

type TestExecutionCompleted struct {
	BaseDomainEvent
	ExecutionID   string                 `json:"execution_id"`
	SuiteID       string                 `json:"suite_id"`
	Results       map[string]interface{} `json:"results"`
	Coverage      map[string]interface{} `json:"coverage"`
	Duration      time.Duration          `json:"duration"`
	PassCount     int                    `json:"pass_count"`
	FailCount     int                    `json:"fail_count"`
}

// QInfra Events
type InfrastructureProvisionRequested struct {
	BaseDomainEvent
	RequestID   string                 `json:"request_id"`
	TenantID    TenantID               `json:"tenant_id"`
	Provider    string                 `json:"provider"`
	Region      string                 `json:"region"`
	Resources   []map[string]interface{} `json:"resources"`
	Specification map[string]interface{} `json:"specification"`
}

type InfrastructureProvisioned struct {
	BaseDomainEvent
	RequestID   string                 `json:"request_id"`
	Resources   []map[string]interface{} `json:"resources"`
	Duration    time.Duration          `json:"duration"`
	Cost        float64                `json:"cost"`
}

type InfrastructureProvisionFailed struct {
	BaseDomainEvent
	RequestID string `json:"request_id"`
	Error     string `json:"error"`
	Reason    string `json:"reason"`
	Rollback  bool   `json:"rollback"`
}

type GoldenImageCreated struct {
	BaseDomainEvent
	ImageID     string    `json:"image_id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	BaseOS      string    `json:"base_os"`
	Size        int64     `json:"size"`
	Checksum    string    `json:"checksum"`
}

// QSecure Events
type SecurityScanStarted struct {
	BaseDomainEvent
	ScanID      string    `json:"scan_id"`
	ProjectID   ProjectID `json:"project_id"`
	ScanType    string    `json:"scan_type"`
	Target      string    `json:"target"`
}

type VulnerabilityDetected struct {
	BaseDomainEvent
	ScanID          string                 `json:"scan_id"`
	VulnerabilityID string                 `json:"vulnerability_id"`
	Severity        string                 `json:"severity"`
	Type            string                 `json:"type"`
	Location        string                 `json:"location"`
	Details         map[string]interface{} `json:"details"`
	Remediation     string                 `json:"remediation"`
}

type SecurityScanCompleted struct {
	BaseDomainEvent
	ScanID            string    `json:"scan_id"`
	Duration          time.Duration `json:"duration"`
	VulnerabilitiesFound int    `json:"vulnerabilities_found"`
	CriticalCount     int       `json:"critical_count"`
	HighCount         int       `json:"high_count"`
	MediumCount       int       `json:"medium_count"`
	LowCount          int       `json:"low_count"`
}

// QSRE Events
type IncidentDetected struct {
	BaseDomainEvent
	IncidentID  string                 `json:"incident_id"`
	TenantID    TenantID               `json:"tenant_id"`
	Service     string                 `json:"service"`
	Severity    string                 `json:"severity"`
	Description string                 `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
}

type IncidentResolved struct {
	BaseDomainEvent
	IncidentID     string        `json:"incident_id"`
	ResolutionTime time.Duration `json:"resolution_time"`
	Resolution     string        `json:"resolution"`
	RootCause      string        `json:"root_cause"`
}

type SLOViolated struct {
	BaseDomainEvent
	SLOID      string                 `json:"slo_id"`
	Service    string                 `json:"service"`
	Metric     string                 `json:"metric"`
	Threshold  float64                `json:"threshold"`
	Actual     float64                `json:"actual"`
	Duration   time.Duration          `json:"duration"`
	Context    map[string]interface{} `json:"context"`
}

// Utility functions for event serialization
func SerializeEvent(event DomainEvent) ([]byte, error) {
	return json.Marshal(event)
}

func DeserializeEvent(data []byte, event interface{}) error {
	return json.Unmarshal(data, event)
}

// Event registry for type mapping
var EventRegistry = map[string]func() DomainEvent{
	"TenantCreated":                    func() DomainEvent { return &TenantCreated{} },
	"TenantUpdated":                    func() DomainEvent { return &TenantUpdated{} },
	"TenantDeactivated":                func() DomainEvent { return &TenantDeactivated{} },
	"UserRegistered":                   func() DomainEvent { return &UserRegistered{} },
	"UserUpdated":                      func() DomainEvent { return &UserUpdated{} },
	"UserDeactivated":                  func() DomainEvent { return &UserDeactivated{} },
	"ProjectCreated":                   func() DomainEvent { return &ProjectCreated{} },
	"ProjectUpdated":                   func() DomainEvent { return &ProjectUpdated{} },
	"ProjectDeleted":                   func() DomainEvent { return &ProjectDeleted{} },
	"WorkspaceCreated":                 func() DomainEvent { return &WorkspaceCreated{} },
	"WorkspaceUpdated":                 func() DomainEvent { return &WorkspaceUpdated{} },
	"WorkspaceDeleted":                 func() DomainEvent { return &WorkspaceDeleted{} },
	"CodeGenerationRequested":          func() DomainEvent { return &CodeGenerationRequested{} },
	"CodeGenerated":                    func() DomainEvent { return &CodeGenerated{} },
	"CodeGenerationFailed":             func() DomainEvent { return &CodeGenerationFailed{} },
	"TestSuiteCreated":                 func() DomainEvent { return &TestSuiteCreated{} },
	"TestsGenerated":                   func() DomainEvent { return &TestsGenerated{} },
	"TestExecutionStarted":             func() DomainEvent { return &TestExecutionStarted{} },
	"TestExecutionCompleted":           func() DomainEvent { return &TestExecutionCompleted{} },
	"InfrastructureProvisionRequested": func() DomainEvent { return &InfrastructureProvisionRequested{} },
	"InfrastructureProvisioned":        func() DomainEvent { return &InfrastructureProvisioned{} },
	"InfrastructureProvisionFailed":    func() DomainEvent { return &InfrastructureProvisionFailed{} },
	"GoldenImageCreated":               func() DomainEvent { return &GoldenImageCreated{} },
	"SecurityScanStarted":              func() DomainEvent { return &SecurityScanStarted{} },
	"VulnerabilityDetected":            func() DomainEvent { return &VulnerabilityDetected{} },
	"SecurityScanCompleted":            func() DomainEvent { return &SecurityScanCompleted{} },
	"IncidentDetected":                 func() DomainEvent { return &IncidentDetected{} },
	"IncidentResolved":                 func() DomainEvent { return &IncidentResolved{} },
	"SLOViolated":                      func() DomainEvent { return &SLOViolated{} },
}