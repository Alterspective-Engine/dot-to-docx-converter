# CLAUDE.md - Development Guidelines and Best Practices

This document contains specific instructions and guidelines for Claude to follow when working on this project.

## Core Development Principles

### 1. File Management Rules
- **NEVER create orphaned files** - Every new file must be integrated into the project structure
- **ALWAYS update existing files** before creating new ones - Check if functionality can be added to existing modules
- **Reduce duplication and redundancy** - DRY (Don't Repeat Yourself) principle must be strictly followed
- **Consolidate similar functionality** - Group related functions and components together

### 2. Coding Best Practices
- Follow language-specific conventions (Go conventions for this project)
- Use meaningful variable and function names
- Write self-documenting code with clear intent
- Keep functions small and focused (single responsibility)
- Handle errors properly and consistently
- Add appropriate logging at key points
- Maintain consistent code formatting

## Self-Evaluation Rubric System

Before implementing any change, evaluate the approach using these 12 categories (score 1-10):

### Planning & Analysis
1. **Problem Understanding** - Is the problem/requirement fully understood?
2. **Solution Design** - Is the solution well-architected and thought through?
3. **Impact Analysis** - Have all impacts and side effects been considered?

### Code Quality
4. **Code Clarity** - Is the code clear, readable, and self-explanatory?
5. **Best Practices** - Does it follow established patterns and conventions?
6. **Error Handling** - Are errors properly caught and handled?

### Integration
7. **File Organization** - Are files properly organized without orphans?
8. **Redundancy Check** - Is there no duplication of existing functionality?
9. **Dependencies** - Are dependencies minimal and well-managed?

### Testing & Validation
10. **Test Coverage** - Is the code testable and are tests considered?
11. **Edge Cases** - Have edge cases and error scenarios been addressed?
12. **Performance** - Is the solution efficient and scalable?

**Requirement**: Any category scoring below 8 must be improved before proceeding.

## Implementation Process

### Phase 1: Planning
1. Create a detailed plan using TodoWrite tool
2. Break down the task into specific, actionable items
3. Identify potential challenges and dependencies
4. Review existing code to avoid duplication

### Phase 2: Implementation
1. Mark each todo item as "in_progress" when starting
2. Follow the plan systematically
3. Check off completed items immediately
4. Document any deviations from the plan

### Phase 3: Review & Reflection
After each code change, ALWAYS:
1. Review what was just implemented
2. Reflect on whether it meets the requirements
3. Check for any unintended side effects
4. Verify integration with existing code
5. Update documentation if needed

## Git Best Practices

### Commit Guidelines
- Make atomic commits - one logical change per commit
- Write clear, descriptive commit messages:
  - First line: Brief summary (50 chars max)
  - Blank line
  - Detailed explanation if needed
- Never commit broken code
- Always review changes before committing

### Branch Management
- Work on feature branches when applicable
- Keep commits organized and logical
- Squash related commits when appropriate
- Ensure main/master branch remains stable

### Before Committing
1. Run linters: `go fmt`, `go vet`, `golint`
2. Run tests if available
3. Review all changes with `git diff`
4. Ensure no sensitive information is included
5. Verify no orphaned or temporary files are included

## Project-Specific Guidelines

### For This Project (DOT to DOCX Converter)
- Maintain separation between API handlers, business logic, and infrastructure
- Keep the converter interface abstract for future implementations
- Ensure proper error messages for API responses
- Maintain consistent logging throughout the application
- Follow Go idioms and conventions
- Use meaningful package names and organization

### Testing Commands
Always run these before finalizing changes:
```bash
go fmt ./...
go vet ./...
go test ./...
go build
```

## Workflow Checklist

For every user instruction:

- [ ] Understand the requirement completely
- [ ] Apply the 12-category rubric evaluation
- [ ] Create a detailed plan with TodoWrite
- [ ] Check for existing similar functionality
- [ ] Implement following best practices
- [ ] Review and reflect on changes
- [ ] Run validation commands
- [ ] Update documentation if needed
- [ ] Prepare clear commit message
- [ ] Verify no orphaned files created

## Notes for Future Sessions

- This project uses LibreOffice for conversion
- Azure Storage integration is optional (falls back to local)
- Redis is optional for queue (falls back to in-memory)
- Synchronous endpoints exist at `/api/v1/convert/sync`
- Maximum file size for sync: 10MB (configurable)
- Worker pool size is configurable via environment variables

## Azure Deployment Information

**IMPORTANT: Deploy to AUSTRALIA EAST for better performance:**

### Australia East Deployment (RECOMMENDED)
- **Resource Group**: `DocSpective`
- **Container App Name**: `dot-to-docx-converter-au`
- **Container Apps Environment**: `dot-to-docx-converter-au-env`
- **Container Registry**: `alterspectiveacr.azurecr.io`
- **Region**: `australiaeast`
- **Deployment Script**: `deploy-to-azure-australia.ps1`

### Legacy East US Deployment (Existing)
- **Container App Name**: `dot-to-docx-converter-prod`
- **Container App URL**: `https://dot-to-docx-converter-prod.lemondesert-9ded9ffc.eastus.azurecontainerapps.io`
- **Region**: East US
- **Container Apps Environment**: `dot-to-docx-converter-prod-env`
- **Note**: This deployment exists but should be migrated to Australia East

## Reflection Questions

After each session, consider:
1. Were any redundant files or functions created?
2. Could the solution be simplified further?
3. Are there any code smells that need addressing?
4. Is the code maintainable by others?
5. Have all edge cases been considered?
6. Is the documentation up to date?
7. Are commit messages clear and helpful?

---

*Last Updated: 2025-09-18*
*This file should be reviewed and updated regularly to maintain relevance.*