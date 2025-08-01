# Execute PRP

Implement a feature using the PRP file with validation and analysis.

## PRP: $ARGUMENTS

## Execution Process

The Agent has Web search capabilities, so passes urls to documentation and examples, the Github tools capabilities, so can search for the AI agent to use.

1. **Load PRP**
    - Read the specified PRP file from ./.claude/planning/$ARGUMENTS.md
    - Understand all context and requirements
    - Follow all instructions in the PRP and extend the research if needed
    - Ensure you have all necessary context to implement the PRP fully
    - Do more web searches and codebase exploration as needed

2. **ULTRA THINK**
    - Think hard before you execute the plan. Create a comprehensive plan addressing all requirements.
    - Break down complex tasks into smaller, manageable steps using your todos tools.
    - Use the TodoWrite tool to create and track your implementation plan.
    - Identify implementation patterns from existing code to follow.
    - **Plan comprehensive test implementation strategy following project patterns**

3. **Execute the plan**
    - Execute the PRP following hexagonal architecture principles
    - Implement all production code following existing patterns
    - **Implement comprehensive test suite** (CRITICAL - not optional):
      - Unit tests for all new functions/types using Go testing and table-driven tests
      - Integration tests for complete feature flows using Testcontainers for Go
      - Mock verification using testify/mock patterns established in codebase  
      - Test data builders using existing helpers or creating new ones
      - JSON marshaling/unmarshaling tests for structs and events
      - Error handling and edge case testing
    - Run `go fmt ./...` to format the code

4. **Validate**
   - **ALWAYS run `go fmt ./...`** first to ensure code formatting compliance
    - Run build command: `make build`
    - **Run comprehensive test suites**:
      - Unit tests: `go test ./... -v -race -coverprofile=coverage.out`
      - Integration tests: `go test ./... -tags=integration -v`
      - Linting and security: `go vet ./... && golangci-lint run`
      - **Verify test coverage meets project standards**
      - **Confirm all test patterns follow established Go conventions**
    - Fix any failures following project debugging patterns
    - Re-run until all passes with proper test coverage

5. **Complete**
    - Ensure all checklist items done
    - Run the final validation suite
    - Report completion status
    - Read the PRP again to ensure you have implemented everything

6. **Post-Execution Analysis**
   - Collect implementation metrics
   - Analyze success and failure patterns
   - **Analyze test coverage and quality metrics**
   - **Document any new testing patterns discovered**
   - Update the knowledge base with learnings or new implementations (at ./docs directory)
   - Generate improvement recommendations
   - Update template suggestions
   - Update the PRP template with any new patterns or insights gained during execution
   - **Update testing documentation if new patterns emerged**

7. **Reference the PRP**
    - You can always reference the PRP again if needed
    - Update the Jira issue with the implementation details
    - **Document test implementation details and coverage achieved**

## Critical Testing Implementation Guidelines

### Mandatory Test Implementation Steps

1. **Unit Test Creation** (NEVER skip these):
   ```go
   func TestNewFeature(t *testing.T) {
       tests := []struct {
           name     string
           input    InputType
           expected ExpectedType
           wantErr  bool
       }{
           {
               name:     "should perform expected behavior",
               input:    testutil.CreateInput(),
               expected: expectedResult,
               wantErr:  false,
           },
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Given
               mockService := new(MockServiceType)
               mockService.On("Method", mock.AnythingOfType("InputType")).Return(expectedResult, nil)
               
               // When
               result, err := featureUnderTest.Execute(tt.input)
               
               // Then
               if tt.wantErr {
                   assert.Error(t, err)
               } else {
                   assert.NoError(t, err)
                   assert.Equal(t, tt.expected, result)
               }
               mockService.AssertExpectations(t)
           })
       }
   }
   ```

2. **Integration Test Requirements**:
   - Must use build tags `// +build integration` to separate from unit tests
   - Must test complete end-to-end scenarios
   - Must verify external system interactions (Kafka, Database)
   - Must include error handling scenarios

3. **Test Data Management**:
   - Use existing builders from `pkg/testutil/` package
   - Create new test data builders following established patterns if needed
   - Ensure test data isolation and cleanup

4. **Mock Verification Patterns**:
   ```go
   type MockService struct {
       mock.Mock
   }
   
   func (m *MockService) Method(arg ArgumentType) (ResultType, error) {
       args := m.Called(arg)
       return args.Get(0).(ResultType), args.Error(1)
   }
   
   // In test:
   mockService := new(MockService)
   mockService.On("Method", mock.MatchedBy(func(arg ArgumentType) bool {
       return arg.Property == expectedValue
   })).Return(expectedResult, nil)
   
   // Execute test
   
   mockService.AssertExpectations(t)
   mockService.AssertNumberOfCalls(t, "Method", 1)
   ```

### Test Coverage Requirements

- **Minimum Coverage**: All new production code must have unit tests
- **Integration Coverage**: All new external integrations must have integration tests  
- **Error Path Coverage**: All error scenarios must be tested
- **Serialization Testing**: All structs/Events must have JSON marshaling tests
- **Code Quality**: New code must pass golangci-lint checks and go vet

### Test Execution Validation

```bash
# MANDATORY validation sequence (must pass before PR):
go fmt ./...                                    # Format code first
go test ./... -v -race -coverprofile=coverage.out  # Run all unit tests with coverage
go test ./... -tags=integration -v              # Run integration tests
go vet ./...                                    # Static analysis
golangci-lint run                               # Comprehensive linting
make build                                      # Full build with all checks
```

**FAILURE IS NOT ACCEPTABLE**: If any test fails, implementation must be fixed before completion.

Note: If validation fails, use error patterns in PRP to fix and retry.