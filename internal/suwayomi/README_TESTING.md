# Suwayomi Client Testing Guide

## Test Types

This package contains two types of tests:

### 1. Unit Tests (Mock Server)
**Files:** `graphql_test.go`, `extensions_test.go`

**What they test:**
- ✅ Client code logic and structure
- ✅ Request construction (headers, body, method)
- ✅ Response parsing
- ✅ Error handling

**What they DON'T test:**
- ❌ Actual Suwayomi API compatibility
- ❌ Real GraphQL query/mutation correctness
- ❌ Field name accuracy

**Run with:**
```bash
go test ./internal/suwayomi/... -v
```

### 2. Integration Tests (Real Server)
**Files:** `integration_test.go`

**What they test:**
- ✅ Actual API compatibility
- ✅ Real server responses
- ✅ GraphQL query correctness
- ✅ Field names and structure

**Requirements:**
- A running Suwayomi-Server instance
- Set `SUWAYOMI_URL` environment variable

**Run with:**
```bash
# Start your Suwayomi-Server first
export SUWAYOMI_URL="http://localhost:4567"
go test -tags=integration -v ./internal/suwayomi/...
```

## Confidence Assessment

### Unit Tests Confidence: **Medium** 🟡
The unit tests verify that the code works **as written**, but since they use mocked responses, they can't guarantee the code matches the real Suwayomi API. The GraphQL queries, field names, and response structures were inferred from the existing codebase but have not been validated against an actual server.

**Known Limitations:**
1. Mock responses might not match real API structure
2. GraphQL queries might have syntax errors
3. Field names (e.g., `thumbnailUrl`, `chapterNumber`) might be incorrect
4. API endpoint paths might be wrong

### Integration Tests Confidence: **High** ✅
When run against a real server, these tests will definitively prove whether the API integration works correctly.

## How to Validate API Correctness

### Method 1: Use GraphiQL (Recommended)
1. Start Suwayomi-Server
2. Open browser to `http://localhost:4567/api/graphql`
3. Use GraphiQL IDE to test queries
4. Compare with our code

### Method 2: Run Integration Tests
```bash
# Ensure server is running
export SUWAYOMI_URL="http://localhost:4567"
go test -tags=integration -v ./internal/suwayomi/...
```

### Method 3: Query Schema via Introspection
```graphql
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    types {
      name
      kind
      fields {
        name
        type {
          name
          kind
        }
      }
    }
  }
}
```

## Potential Issues to Check

If integration tests fail, check these common issues:

1. **GraphQL Query Syntax**
   - Field names might be incorrect
   - Query structure might not match schema

2. **Response Field Names**
   - Our code expects: `thumbnailUrl`, `chapterNumber`, `isRead`
   - Server might use: `thumbnail_url`, `chapter_number`, `read`

3. **Type Mismatches**
   - Integer vs String for IDs
   - Float vs Int for chapter numbers
   - Timestamp formats

4. **API Version**
   - Suwayomi v1.0.0 redid the entire API
   - Older servers might have different schema

## Test Coverage Summary

| Component | Unit Tests | Integration Tests |
|-----------|-----------|-------------------|
| GraphQL Client | ✅ 15 tests | ✅ 6 tests |
| Extension Management | ✅ 9 tests | ✅ 2 tests |
| Server Health | ✅ 8 tests | ✅ 1 test |
| Manga Operations | ✅ 5 tests | ✅ 3 tests |
| Chapter Operations | ✅ 3 tests | ✅ 2 tests |

## Recommendations

1. **Run integration tests** against a real Suwayomi-Server to validate API correctness
2. **Check GraphiQL** at `/api/graphql` to explore the actual schema
3. **Update tests** if field names or query structures don't match
4. **Document findings** - if integration tests reveal issues, update this README

## Example Integration Test Output

```
=== RUN   TestIntegration_HealthCheck
    integration_test.go:32: Server Version: v1.0.0
    integration_test.go:33: Build Type: Stable
    integration_test.go:34: Revision: abc123
    integration_test.go:35: Extension Count: 12
    integration_test.go:36: Manga Count: 342
--- PASS: TestIntegration_HealthCheck (0.15s)
```

## Contributing

If you find that the tests don't match the real Suwayomi API:

1. Run integration tests to identify specific failures
2. Check actual API response using GraphiQL
3. Update the client code to match real API
4. Update unit test mocks to match real responses
5. Document the changes in git commit

## Resources

- [Suwayomi-Server GitHub](https://github.com/Suwayomi/Suwayomi-Server)
- [GraphQL Introspection](https://graphql.org/learn/introspection/)
- [Suwayomi v1.0.0 Release Notes](https://github.com/Suwayomi/Suwayomi-Server/releases/tag/v1.0.0)
