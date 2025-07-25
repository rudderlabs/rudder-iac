# Pull Request #126: feat: new command to list retl sources supports only sql models

## Description of the change

This PR implements a new CLI command `rudder-cli workspace retl-source list` that allows users to list RETL SQL model sources from their workspace. This is the first phase of implementing WAR-944, focusing specifically on listing remote SQL models.

### Key Features:
- **New CLI Command**: `rudder-cli workspace retl-source list` with support for both table and JSON output formats
- **Provider Extension**: Extended the RETL provider to support listing operations through a new `List` method
- **Handler Implementation**: Added `List` method to the SQL model handler that fetches and transforms RETL sources from the API
- **Comprehensive Testing**: Added extensive test coverage for all new functionality including success, error, and edge cases

### Technical Implementation:
- Added new command structure under `workspace` hierarchy following Cobra command patterns
- Implemented provider-level listing interface that delegates to resource-specific handlers
- Enhanced SQL model handler with listing capability that properly formats response data
- Added telemetry tracking for the new command
- Updated API client with clearer documentation

## Type of change
- [x] New feature (non-breaking change that adds functionality)
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)

## Related issues

> Fixes WAR-944 (Phase 1: Implement list remote SQL models command)

## Checklists

### Development

- [x] Lint rules pass locally
- [x] The code changed/added as part of this pull request has been covered with tests
- [x] All tests related to the changed code pass in development

### Code review 

- [x] This pull request has a descriptive title and information useful to a reviewer
- [ ] "Ready for review" label attached to the PR and reviewers mentioned in a comment
- [ ] Changes have been reviewed by at least one other engineer
- [ ] Issue from task tracker has a link to this pull request

## Testing Notes

The PR includes comprehensive test coverage:
- Unit tests for the new CLI command
- Provider-level tests for the List method
- Handler tests covering success, empty results, and error scenarios
- Integration tests ensuring proper data transformation and formatting

## Usage Example

```bash
# List RETL sources in table format (default)
rudder-cli workspace retl-source list

# List RETL sources in JSON format
rudder-cli workspace retl-source list --json
```

## Future Work

This is Phase 1 of WAR-944. Future phases may include:
- Support for additional RETL source types beyond SQL models
- Filtering and search capabilities
- Enhanced output formatting options 