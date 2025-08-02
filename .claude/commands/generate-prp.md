# Generate PRP (Product Requirements Prompts) for Feature Implementation

## GitHub Issue: $ARGUMENTS

Generate a complete PRP for general feature implementation with thorough research. Ensure context is passed to the AI agent to enable self-validation and iterative refinement. Read the feature GitHub issue first to understand what needs to be created, how the examples provided help, and any other considerations.

The AI agent only gets the context you are appending to the PRP and training data. Assume the AI agent has access to the codebase and the same knowledge cutoff as you, so it's important that your research findings are included or referenced in the PRP.

The Agent has web search capabilities, so pass URLs to documentation and examples, and have github issue access capabilities for comprehensive research.

## Research Process

1. **Read the Feature GitHub Issue**
    - Use `hub` command line if available
    - Understand the feature requirements
    - Identify examples provided in the issue
    - Note any specific patterns or conventions mentioned

2. **Codebase Analysis**
    - Search for similar features/patterns in the codebase
    - Identify files to reference in PRP
    - Note existing conventions to follow
    
3. **Test Pattern Analysis** - Critical for implementation success:
   - Analyze existing test structure and naming conventions
   - Identify test frameworks used (Go testing, testify, Testcontainers for Go)
   - Review integration test patterns for similar features
   - Check test data factory patterns in existing test files
   - Examine mocking patterns and verification strategies (testify/mock, gomock)
   - Review test organization patterns (table-driven tests, subtests)

4. **External Research**
    - Search for similar features/patterns online
    - Library documentation (include specific URLs)
    - Implementation examples (GitHub/StackOverflow/blogs)
    - Best practices and common pitfalls

5. **User Clarification** (if needed)
    - Specific patterns to mirror and where to find them?
    - Integration requirements and where to find them?

## PRP Generation

### Critical Context to Include and Pass to the AI Agent as Part of the PRP
- **Documentation**: URLs with specific sections
- **Code Examples**: Real snippets from codebase
- **Gotchas**: Library quirks, version issues
- **Patterns**: Existing approaches to follow

### Implementation Blueprint
- Start with pseudocode showing approach
- Reference real files for patterns
- Include error handling strategy
- List tasks to be completed to fulfill the PRP in the order they should be completed

### Testing Strategy (CRITICAL - Must be Comprehensive)

**Unit Test Requirements:**
- Use standard Go testing framework with `testing.T` and table-driven tests
- Follow Go conventions with `TestFunctionName` format and subtests using `t.Run()`
- Use testify/assert and testify/mock for assertions and mocking
- Include JSON marshaling/unmarshaling tests for structs and events
- Test data transformation and business logic thoroughly
- Use test helper functions and builders for consistent test data generation

**Integration Test Requirements:**  
- Use build tags (`// +build integration`) to separate integration tests
- Use Testcontainers for Go to spin up real database and message broker instances
- Test complete end-to-end flows with actual infrastructure components
- Verify event publishing to correct topics/queues with real event bus
- Test error handling, retries, and graceful degradation scenarios

**Test File Structure:**
```
pkg/
├── models/workflow_test.go                         # Unit tests for domain models
├── actions/http_request/action_test.go             # Unit tests for actions
├── triggers/webhook/trigger_test.go                # Unit tests for triggers
├── workflow/engine_test.go                         # Unit tests for workflow engine
├── event_bus/kafka_integration_test.go             # Integration tests with build tags
└── testutil/builders.go                            # Test data builders and helpers
```

**Required Test Coverage:**
- Struct construction and field validation
- JSON marshaling/unmarshaling with correct struct tags
- Data transformation and business logic accuracy  
- Event bus message publishing verification
- Error handling and graceful degradation
- Idempotency and duplicate handling
- Performance benchmarks and timing validations

### Validation Gates (Must be Executable)
```bash
# Code Formatting (CRITICAL - always run first)
go fmt ./...

# Syntax/Style/Security checks
go vet ./...
golangci-lint run

# Unit Tests (with coverage)
go test ./... -v -race -coverprofile=coverage.out

# Integration Tests (with build tags)
go test ./... -tags=integration -v

# Benchmarks and performance tests
go test ./... -bench=. -benchmem

# Full build with all checks
make build && make test
```

*** CRITICAL: Complete all research and codebase exploration before writing the PRP ***

*** ULTRA THINK ABOUT THE PRP AND PLAN YOUR APPROACH THEN START WRITING THE PRP ***

## Output

The output should be saved as a file at "./.claude/planning/{{issue number}}.md".

## Quality Checklist
- [ ] All necessary context included
- [ ] Validation gates are executable by AI
- [ ] References to existing patterns
- [ ] Clear implementation path
- [ ] Error handling documented
- [ ] **Comprehensive test strategy documented**
- [ ] **Test framework patterns and examples included**
- [ ] **Unit and integration test requirements specified**
- [ ] **Test data factory patterns referenced**
- [ ] **Mocking strategies and verification patterns documented**
- [ ] **Coverage expectations clearly defined**

Score the PRP on a scale of 1–10 (confidence level to succeed in one-pass implementation using Claude Code)

### Testing Documentation Requirements

The PRP must include specific testing guidance:

1. **Test Framework Setup Examples**
   ```go
   func TestFeatureName(t *testing.T) {
       tests := []struct {
           name     string
           input    interface{}
           expected interface{}
           wantErr  bool
       }{
           {
               name:     "should behave correctly",
               input:    testInput,
               expected: expectedOutput,
               wantErr:  false,
           },
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Given-When-Then pattern implementation
           })
       }
   }
   ```

2. **Testify Mock Usage Patterns**
   ```go
   type MockService struct {
       mock.Mock
   }
   
   func (m *MockService) Method(arg ArgumentType) error {
       args := m.Called(arg)
       return args.Error(0)
   }
   
   // In test:
   mockService := new(MockService)
   mockService.On("Method", mock.AnythingOfType("ArgumentType")).Return(nil)
   mockService.AssertExpectations(t)
   ```

3. **Integration Test Template**
   ```go
   // +build integration
   
   func TestFeatureIntegration(t *testing.T) {
       // Testcontainer setup
       ctx := context.Background()
       container, err := testcontainers.GenericContainer(ctx, ...)
       require.NoError(t, err)
       defer container.Terminate(ctx)
       
       // Test implementation
   }
   ```

4. **Test Data Generation**
   - Reference existing builders in `pkg/testutil/`
   - Pattern for creating new test data builders
   - Usage examples with structured test data generation

Remember: The goal is one-pass implementation success through a comprehensive context.