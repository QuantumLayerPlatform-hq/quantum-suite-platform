# Contract Validation Framework

## Overview

The Contract Validation Framework ensures all services and workflows comply with their defined contracts, enabling **deterministic interoperability** across the Quantum Suite platform.

## Validation Types

### 1. 🔧 Schema Validation
- **API Request/Response Validation** - Ensure all HTTP APIs conform to OpenAPI specs
- **Event Schema Validation** - Validate domain events against JSON schemas
- **Workflow Input/Output Validation** - Check workflow parameters and results
- **QLAFS Data Validation** - Validate fingerprint and provenance data structures

### 2. 🔄 Contract Compliance Testing
- **Consumer Contract Tests** - Services validate their dependencies
- **Provider Contract Tests** - Services validate their interfaces  
- **Integration Contract Tests** - End-to-end workflow validation
- **Mock Service Generation** - Auto-generated contract-compliant mocks

### 3. 🚦 Runtime Validation
- **API Gateway Validation** - Real-time request/response validation
- **Event Bus Validation** - Event schema validation on publish/consume
- **Workflow Engine Validation** - Workflow input/output validation
- **QLAFS Validation** - Fingerprint and provenance data validation

## Implementation Structure

```go
// Contract validation interface
type ContractValidator interface {
    ValidateRequest(service string, operation string, data interface{}) error
    ValidateResponse(service string, operation string, data interface{}) error
    ValidateEvent(eventType string, data interface{}) error
    ValidateWorkflow(workflowType string, input interface{}, output interface{}) error
}

// Schema-based validator implementation
type SchemaValidator struct {
    apiSchemas      map[string]*openapi3.T
    eventSchemas    map[string]*jsonschema.Schema
    workflowSchemas map[string]*WorkflowContract
}

func (v *SchemaValidator) ValidateRequest(service string, operation string, data interface{}) error {
    schema, ok := v.apiSchemas[service]
    if !ok {
        return fmt.Errorf("no API schema found for service: %s", service)
    }
    
    // Validate against OpenAPI schema
    return validateOpenAPI(schema, operation, data)
}

func (v *SchemaValidator) ValidateEvent(eventType string, data interface{}) error {
    schema, ok := v.eventSchemas[eventType]
    if !ok {
        return fmt.Errorf("no event schema found for type: %s", eventType)
    }
    
    // Validate against JSON schema
    return schema.Validate(data)
}
```

## Contract Test Examples

### API Contract Test
```go
func TestTemporalWorkflowsAPIContract(t *testing.T) {
    // Load contract specification
    contract := LoadAPIContract("temporal-workflows")
    
    // Test valid request
    validRequest := StartWorkflowRequest{
        WorkflowType: "CodeGenerationPipeline",
        Input: map[string]interface{}{
            "request_id": "test-123",
            "prompt":     "Create a REST API",
            "language":   "go",
        },
    }
    
    err := contract.ValidateRequest("startWorkflow", validRequest)
    assert.NoError(t, err)
    
    // Test invalid request
    invalidRequest := StartWorkflowRequest{
        WorkflowType: "InvalidType", // Not in enum
        Input:        map[string]interface{}{},
    }
    
    err = contract.ValidateRequest("startWorkflow", invalidRequest)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "InvalidType")
}
```

### Event Contract Test  
```go
func TestWorkflowEventsContract(t *testing.T) {
    // Load event contract
    contract := LoadEventContract("workflow-events")
    
    // Test valid WorkflowStarted event
    validEvent := map[string]interface{}{
        "event_id":      "550e8400-e29b-41d4-a716-446655440000",
        "event_type":    "WorkflowStarted",
        "workflow_id":   "wf-123",
        "workflow_type": "CodeGenerationPipeline",
        "timestamp":     "2025-01-15T10:30:00Z",
        "version":       1,
        "input": map[string]interface{}{
            "prompt":   "Create function",
            "language": "go",
        },
    }
    
    err := contract.ValidateEvent("WorkflowStarted", validEvent)
    assert.NoError(t, err)
    
    // Test missing required field
    invalidEvent := map[string]interface{}{
        "event_id":   "550e8400-e29b-41d4-a716-446655440000",
        "event_type": "WorkflowStarted",
        // Missing workflow_id - should fail
        "timestamp": "2025-01-15T10:30:00Z",
        "version":   1,
    }
    
    err = contract.ValidateEvent("WorkflowStarted", invalidEvent)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "workflow_id")
}
```

### Workflow Contract Test
```go
func TestCodeGenerationPipelineContract(t *testing.T) {
    // Load workflow contract
    contract := LoadWorkflowContract("CodeGenerationPipeline")
    
    // Test valid input
    validInput := map[string]interface{}{
        "request_id": "550e8400-e29b-41d4-a716-446655440000",
        "prompt":     "Create a function to validate email addresses",
        "language":   "go",
        "options": map[string]interface{}{
            "include_tests":         true,
            "include_security_scan": false,
        },
    }
    
    err := contract.ValidateInput(validInput)
    assert.NoError(t, err)
    
    // Test valid output
    validOutput := map[string]interface{}{
        "request_id": "550e8400-e29b-41d4-a716-446655440000",
        "status":     "completed",
        "generated_code": map[string]interface{}{
            "content":    "func ValidateEmail(email string) bool { ... }",
            "language":   "go",
            "confidence": 0.95,
            "iterations": 2,
        },
        "fingerprint_record": map[string]interface{}{
            "fingerprint_id":   "fp-123",
            "agent_id":         "qagent-v1.2.3",
            "fingerprint_hash": "sha256:abc123...",
            "confidence":       0.91,
        },
        "provenance_record": map[string]interface{}{
            "record_id":    "prov-456",
            "operation_id": "op-789",
            "merkle_proof": []string{"hash1", "hash2"},
        },
    }
    
    err = contract.ValidateOutput(validOutput)
    assert.NoError(t, err)
}
```

### QLAFS Contract Test
```go
func TestAgentFingerprintContract(t *testing.T) {
    // Load QLAFS fingerprint contract
    contract := LoadQLAFSContract("agent-fingerprint")
    
    // Test valid fingerprint
    validFingerprint := map[string]interface{}{
        "agent_id":         "qagent-v1.2.3",
        "agent_type":       "code_generation",
        "fingerprint_hash": "sha256:a3b5c7d9e1f2a4b6c8d0e2f4a6b8c0d2e4f6a8b0c2d4e6f8a0b2c4d6e8f0a2b4",
        "dimensions": map[string]interface{}{
            "static": map[string]interface{}{
                "model_architecture": "transformer",
                "parameter_count":     175000000000,
                "weights_hash":        "sha256:b4c6d8e0f2a4...",
                "model_version":       "1.2.3",
            },
            "behavioral": map[string]interface{}{
                "response_patterns": []map[string]interface{}{
                    {
                        "input_class":  "rest_api_request",
                        "output_class": "go_handler_function",
                        "confidence":   0.95,
                        "frequency":    150,
                    },
                },
                "decision_consistency": 0.92,
            },
        },
        "confidence": 0.91,
        "created_at": "2025-01-15T10:30:00Z",
    }
    
    err := contract.ValidateFingerprint(validFingerprint)
    assert.NoError(t, err)
    
    // Test invalid agent_type
    invalidFingerprint := validFingerprint
    invalidFingerprint["agent_type"] = "invalid_type"
    
    err = contract.ValidateFingerprint(invalidFingerprint)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "agent_type")
}
```

## Mock Service Generation

### Auto-Generated Mocks
```go
// Auto-generated mock from Temporal workflows contract
type MockTemporalWorkflowsService struct {
    contract *APIContract
}

func (m *MockTemporalWorkflowsService) StartWorkflow(req StartWorkflowRequest) (*WorkflowExecution, error) {
    // Validate request against contract
    if err := m.contract.ValidateRequest("startWorkflow", req); err != nil {
        return nil, err
    }
    
    // Generate deterministic response based on contract
    response := &WorkflowExecution{
        WorkflowID:   generateDeterministicUUID(req.WorkflowType, req.Input),
        RunID:        generateDeterministicUUID("run", req),
        WorkflowType: req.WorkflowType,
        Status:       "RUNNING",
        CreatedAt:    time.Now(),
        Input:        req.Input,
    }
    
    // Validate response against contract
    if err := m.contract.ValidateResponse("startWorkflow", response); err != nil {
        return nil, fmt.Errorf("mock generated invalid response: %w", err)
    }
    
    return response, nil
}

// Deterministic UUID generation for predictable testing
func generateDeterministicUUID(prefix string, data interface{}) string {
    hasher := sha256.New()
    hasher.Write([]byte(prefix))
    hasher.Write([]byte(fmt.Sprintf("%v", data)))
    hash := hasher.Sum(nil)
    
    // Convert hash to UUID format
    return fmt.Sprintf("%x-%x-%x-%x-%x",
        hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}
```

## CI/CD Integration

### Contract Validation Pipeline
```yaml
# .github/workflows/contract-validation.yml
name: Contract Validation
on: [push, pull_request]

jobs:
  validate-contracts:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.21
          
      - name: Install contract tools
        run: |
          go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest
          npm install -g @apidevtools/swagger-parser
          npm install -g ajv-cli
          
      - name: Validate API contracts
        run: |
          make validate-api-contracts
          
      - name: Validate event contracts  
        run: |
          make validate-event-contracts
          
      - name: Validate workflow contracts
        run: |
          make validate-workflow-contracts
          
      - name: Run contract tests
        run: |
          make test-contracts
          
      - name: Generate contract documentation
        run: |
          make docs-contracts
          
      - name: Check breaking changes
        run: |
          make check-breaking-changes

  contract-compatibility:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [temporal, qlafs, llm-gateway, vector-db]
    steps:
      - uses: actions/checkout@v3
      
      - name: Test ${{ matrix.service }} contract compatibility
        run: |
          make test-service-contract service=${{ matrix.service }}
```

### Makefile Targets
```makefile
# Contract validation targets
.PHONY: validate-contracts validate-api-contracts validate-event-contracts validate-workflow-contracts

validate-contracts: validate-api-contracts validate-event-contracts validate-workflow-contracts

validate-api-contracts:
	@echo "Validating API contracts..."
	@for file in docs/contracts/api/**/*.yaml; do \
		swagger-parser validate "$$file" || exit 1; \
	done

validate-event-contracts:
	@echo "Validating event contracts..."
	@for file in docs/contracts/events/**/*.json; do \
		ajv validate -s "$$file" -d /dev/null || exit 1; \
	done

validate-workflow-contracts:
	@echo "Validating workflow contracts..."
	@go run cmd/tools/validate-workflows/main.go docs/contracts/workflows/

test-contracts:
	@echo "Running contract tests..."
	@go test ./tests/contracts/... -v

check-breaking-changes:
	@echo "Checking for breaking changes..."
	@go run cmd/tools/breaking-changes/main.go

docs-contracts:
	@echo "Generating contract documentation..."
	@go run cmd/tools/contract-docs/main.go
```

## Benefits

### ✅ **Deterministic Interoperability**
- **Compile-time Safety** - Contract violations caught before deployment
- **Predictable Behavior** - All services conform to well-defined interfaces
- **Version Compatibility** - Safe evolution with backward compatibility

### 🚀 **Development Velocity**
- **Parallel Development** - Teams work independently with guaranteed integration
- **Reliable Mocks** - Contract-compliant testing without real services
- **Documentation** - Self-documenting APIs from contracts

### 🛡️ **Production Reliability**
- **Runtime Validation** - Real-time contract compliance checking
- **Integration Testing** - Automated end-to-end validation
- **Monitoring** - Contract violation detection and alerting

This comprehensive contract framework ensures the Quantum Suite platform operates with **deterministic interoperability** across all five product lines and shared services! 🎯