package domain

import (
	"time"

	"github.com/google/uuid"
)

// Core entity interfaces that all domain entities must implement
type Entity interface {
	ID() string
	Version() int64
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// AggregateRoot represents the root entity of an aggregate in DDD
type AggregateRoot interface {
	Entity
	Events() []DomainEvent
	ClearEvents()
	ApplyEvent(event DomainEvent) error
}

// DomainEvent represents something that happened in the domain
type DomainEvent interface {
	EventID() string
	EventType() string
	AggregateID() string
	AggregateType() string
	Timestamp() time.Time
	Version() int64
	Metadata() map[string]interface{}
}

// BaseEntity provides common fields for all entities
type BaseEntity struct {
	id        string
	version   int64
	createdAt time.Time
	updatedAt time.Time
}

func NewBaseEntity() BaseEntity {
	now := time.Now()
	return BaseEntity{
		id:        uuid.New().String(),
		version:   1,
		createdAt: now,
		updatedAt: now,
	}
}

func (e BaseEntity) ID() string           { return e.id }
func (e BaseEntity) Version() int64       { return e.version }
func (e BaseEntity) CreatedAt() time.Time { return e.createdAt }
func (e BaseEntity) UpdatedAt() time.Time { return e.updatedAt }

// BaseAggregateRoot provides common functionality for aggregate roots
type BaseAggregateRoot struct {
	BaseEntity
	events []DomainEvent
}

func NewBaseAggregateRoot() BaseAggregateRoot {
	return BaseAggregateRoot{
		BaseEntity: NewBaseEntity(),
		events:     make([]DomainEvent, 0),
	}
}

func (a *BaseAggregateRoot) Events() []DomainEvent {
	return a.events
}

func (a *BaseAggregateRoot) ClearEvents() {
	a.events = make([]DomainEvent, 0)
}

func (a *BaseAggregateRoot) ApplyEvent(event DomainEvent) error {
	a.events = append(a.events, event)
	a.version++
	a.updatedAt = time.Now()
	return nil
}

// Value Objects - Immutable objects that represent descriptive aspects
type (
	TenantID    string
	UserID      string
	ProjectID   string
	WorkspaceID string
	RequestID   string
	AgentID     string
	JobID       string
)

func NewTenantID() TenantID       { return TenantID(uuid.New().String()) }
func NewUserID() UserID           { return UserID(uuid.New().String()) }
func NewProjectID() ProjectID     { return ProjectID(uuid.New().String()) }
func NewWorkspaceID() WorkspaceID { return WorkspaceID(uuid.New().String()) }
func NewRequestID() RequestID     { return RequestID(uuid.New().String()) }
func NewAgentID() AgentID         { return AgentID(uuid.New().String()) }
func NewJobID() JobID             { return JobID(uuid.New().String()) }

// Core Domain Entities

// Tenant represents a customer organization
type Tenant struct {
	BaseAggregateRoot
	Name     string                 `json:"name"`
	Plan     string                 `json:"plan"`
	Status   string                 `json:"status"`
	Settings map[string]interface{} `json:"settings"`
}

// User represents a user within a tenant
type User struct {
	BaseEntity
	TenantID    TenantID               `json:"tenant_id"`
	Email       string                 `json:"email"`
	Name        string                 `json:"name"`
	Role        string                 `json:"role"`
	Preferences map[string]interface{} `json:"preferences"`
}

// Project represents a development project
type Project struct {
	BaseAggregateRoot
	TenantID    TenantID               `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Settings    map[string]interface{} `json:"settings"`
	Status      string                 `json:"status"`
}

// Workspace represents a development workspace within a project
type Workspace struct {
	BaseAggregateRoot
	ProjectID     ProjectID              `json:"project_id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration"`
	State         map[string]interface{} `json:"state"`
}

// Common Enums
type (
	JobStatus string
	Priority  string
)

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"

	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Security Context for authorization
type SecurityContext struct {
	TenantID    TenantID    `json:"tenant_id"`
	UserID      UserID      `json:"user_id"`
	Permissions []string    `json:"permissions"`
	Roles       []string    `json:"roles"`
	MFAVerified bool        `json:"mfa_verified"`
	SessionID   string      `json:"session_id"`
	ExpiresAt   time.Time   `json:"expires_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Audit Log entry for compliance and tracking
type AuditLog struct {
	BaseEntity
	TenantID    TenantID               `json:"tenant_id"`
	UserID      UserID                 `json:"user_id"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource"`
	ResourceID  string                 `json:"resource_id"`
	Changes     map[string]interface{} `json:"changes"`
	IPAddress   string                 `json:"ip_address"`
	UserAgent   string                 `json:"user_agent"`
	Status      string                 `json:"status"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
}

// Business metrics and KPIs
type Metrics struct {
	TenantID    TenantID               `json:"tenant_id"`
	MetricName  string                 `json:"metric_name"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit"`
	Tags        map[string]string      `json:"tags"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}