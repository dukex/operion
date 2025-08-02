# Operion Documentation TODO

This document outlines critical improvements needed for the Operion workflow automation platform documentation based on comprehensive analysis comparing documentation against source code implementation.

## Critical Issues Requiring Immediate Attention

### 1. Template Syntax Corrections (Priority: HIGH)
**Problem**: Documentation uses incorrect template syntax throughout examples
**Impact**: All template examples will fail when users try them
**Files Affected**: 
- `/content/docs/getting-started/developers/templates.mdx`
- `/content/docs/getting-started/developers/first-workflow.mdx`
- Multiple workflow examples

THE CORRECT TEMPLATE IS GO TEMPLATE.

**Required Changes**:
- Replace JSONata syntax (`trigger.body.field`) with Go template syntax (`{{.trigger.body.field}}`)
- Update all 50+ template examples in the comprehensive templates guide
- Fix conditional expressions in workflow examples
- Update transform action examples to use proper Go template syntax

**Verification**: Check against `/pkg/template/template.go` implementation

### 2. Visual Editor Documentation Accuracy (Priority: HIGH)
**Problem**: Documentation overstates current capabilities
**Current State**: Read-only workflow visualization
**Documentation Claims**: Implies editing capabilities

**Required Updates**:
- `/content/docs/visual-editor.mdx` - Clarify read-only nature throughout
- Remove references to "editing" workflows via UI
- Update feature lists to reflect actual implementation
- Add clear roadmap section for planned editing features

### 3. Port Configuration Standardization (Priority: HIGH)
**Problem**: Inconsistent port references across documentation
**Correct Ports** (verified in source):
- API Server: 9091 (default)
- Webhook Server: 8085 (dispatcher service)
- Visual Editor: 5173 (development)

**Files to Update**:
- All tutorial files with localhost URLs
- API reference examples
- Installation guides
- Docker and Kubernetes deployment manifests

### 4. Webhook Trigger Implementation Status (Priority: HIGH)
**Issue**: Need to verify if webhook triggers are fully implemented
**Documentation Status**: Extensive webhook documentation exists
**Required Action**: 
- Verify webhook trigger functionality in source code
- If implemented: Ensure documentation accuracy
- If not implemented: Move to "Planned Features" section

## Missing Documentation (High Priority)

### 5. JSONata vs Go Templates Clarification
**Need**: Clear explanation of template system
**Content Required**:
- Template system overview explaining Go template usage
- Comparison table: what users might expect vs. actual syntax
- Migration guide from other template systems
- Built-in functions reference
- Template debugging guide

### 6. Production Deployment Guide
**Current Gap**: Only development setup covered
**Required Sections**:
- Environment variable configuration for production
- Database setup (PostgreSQL configuration)
- Kafka/message queue production setup
- Load balancing and scaling considerations
- Health check endpoints configuration
- Monitoring and metrics collection

### 7. Error Handling and Debugging
**Missing Content**:
- Common workflow execution errors
- Template evaluation debugging
- Action failure troubleshooting
- Event bus connection issues
- Log analysis guide
- Debug mode configuration

### 8. Complete API Reference
**Current Issues**:
- Incomplete error response examples
- Missing query parameters documentation  
- Webhook endpoint documentation gaps
- Response schema validation rules

**Required Additions**:
- Complete OpenAPI/Swagger specification
- All error scenarios with examples
- Rate limiting information
- Authentication/authorization (when implemented)

## Medium Priority Documentation Needs

### 9. Plugin Development Guide
**Content Needed**:
- Action plugin development tutorial
- Trigger plugin development tutorial
- Plugin testing strategies
- Plugin packaging and deployment
- Schema definition best practices

### 10. Advanced Workflow Patterns
**Missing Examples**:
- Complex conditional logic patterns
- Parallel step execution
- Error handling and retry strategies
- Long-running workflow management
- Data transformation patterns

### 11. Integration Guides
**Needed Integrations**:
- CI/CD pipeline integration
- Monitoring system integration (Prometheus, Grafana)
- External API integration patterns
- Database integration examples
- Message queue integration patterns

### 12. Performance and Scaling
**Documentation Gaps**:
- Performance tuning guide
- Scaling considerations
- Resource usage optimization
- Concurrent workflow handling
- Memory management best practices

## Documentation Structure Improvements

### 13. Enhanced Navigation
**Current Issues**:
- No search functionality mentioned
- Limited cross-referencing
- Missing glossary

**Improvements Needed**:
- Comprehensive glossary of terms
- Better cross-linking between sections
- Quick reference cards
- Troubleshooting decision trees

### 14. Code Example Standardization
**Current Problems**:
- Inconsistent code formatting
- Mixed naming conventions
- Incomplete examples

**Standards Needed**:
- Consistent JSON formatting
- Standard workflow naming patterns
- Complete, runnable examples
- Example validation against actual API

### 15. User Journey Documentation
**Missing Paths**:
- Beginner to advanced progression
- Role-based documentation (operators vs developers)
- Use case specific guides
- Migration scenarios

## Quality Assurance Needs

### 16. Documentation Testing Strategy
**Required**:
- Automated example validation
- Link checking
- Code syntax verification
- Regular documentation reviews

### 17. Feedback and Contribution Guidelines
**Missing**:
- Documentation contribution guide
- Feedback collection mechanism
- Community engagement processes
- Documentation maintenance schedule

## Immediate Action Items

1. **Fix Template Syntax** (Est: 4-6 hours)
   - Update all Go template examples
   - Verify against source code
   - Test example workflows

2. **Clarify Visual Editor Status** (Est: 1-2 hours)
   - Update capability descriptions
   - Add implementation roadmap
   - Remove misleading references

3. **Standardize Port References** (Est: 2-3 hours)
   - Update all localhost URLs
   - Verify in deployment guides
   - Update API examples

4. **Create Production Setup Guide** (Est: 6-8 hours)
   - Database configuration
   - Security considerations
   - Scaling recommendations

5. **Develop Troubleshooting Guide** (Est: 4-5 hours)
   - Common issues compilation
   - Step-by-step solutions
   - Debug configuration guide

## Success Metrics

- All template examples work without modification
- Users can successfully complete tutorials
- Production deployment guide enables real deployments
- Reduced support questions about basic setup
- Improved user onboarding success rate

## Implementation Priority Order

1. **Phase 1** (Week 1): Critical syntax fixes and accuracy issues
2. **Phase 2** (Week 2): Missing core documentation (production, troubleshooting)
3. **Phase 3** (Week 3-4): Advanced guides and integration documentation  
4. **Phase 4** (Ongoing): Quality assurance and maintenance processes

This TODO represents approximately 40-60 hours of documentation work to achieve publication-ready status for a 1.0 release.