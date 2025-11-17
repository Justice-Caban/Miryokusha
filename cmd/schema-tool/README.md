## Suwayomi GraphQL Schema Introspection Tool

This tool uses GraphQL introspection to fetch the complete schema from a running Suwayomi server, allowing you to:

1. **Save the schema** in JSON and GraphQL SDL formats
2. **Validate** your client code against the actual server schema
3. **Detect API changes** before they break your application
4. **Document** the API structure for development

### Building

```bash
cd cmd/schema-tool
go build -o schema-tool
```

### Usage

#### 1. Introspect and Save Schema

```bash
# Basic usage (saves to schema/suwayomi_schema.json and schema/suwayomi_schema.graphql)
./schema-tool -server http://localhost:4567

# Custom output paths
./schema-tool -server http://localhost:4567 \
  -json my_schema.json \
  -sdl my_schema.graphql
```

#### 2. Show Schema Summary

```bash
./schema-tool -server http://localhost:4567 -summary
```

Example output:
```
Suwayomi GraphQL Schema Summary
Server: http://localhost:4567
Info: v1.0.0 (build: Stable, revision: abc123)
Fetched: 2024-11-17T10:30:00Z

Type Counts:
  OBJECT: 45
  ENUM: 12
  INPUT_OBJECT: 18
  INTERFACE: 3
  SCALAR: 8

Query Type: QueryType
Mutation Type: MutationType
Subscription Type: SubscriptionType
```

#### 3. Validate Client Code

```bash
./schema-tool -server http://localhost:4567 -validate
```

This compares your saved baseline schema against the current server schema and reports:
- ❌ **Errors**: Missing types/fields, type mismatches (breaks compatibility)
- ⚠️ **Warnings**: Deprecated fields, optional changes
- ✅ **Success**: Schema is compatible

### Output Files

**JSON Schema** (`schema/suwayomi_schema.json`):
- Complete introspection result
- Machine-readable format
- Used for automated validation
- Includes metadata (fetch time, server info)

**SDL Schema** (`schema/suwayomi_schema.graphql`):
- Human-readable GraphQL schema definition
- Useful for documentation
- Can be used with GraphQL tools
- Shows types, fields, arguments, descriptions

### Integration with Application

The application can automatically validate the schema at startup:

```go
import "github.com/Justice-Caban/Miryokusha/internal/suwayomi"

func main() {
    client := suwayomi.NewClient("http://localhost:4567")

    // Validate schema at startup (respects SUWAYOMI_SCHEMA_VALIDATION env var)
    if err := suwayomi.AutoValidateSchema(client, "schema/suwayomi_schema.json"); err != nil {
        log.Fatalf("Schema validation failed: %v", err)
    }

    // ... rest of application
}
```

### Environment Variables

Control schema validation behavior with `SUWAYOMI_SCHEMA_VALIDATION`:

```bash
# Strict mode: Fail if schema doesn't match (recommended for CI/CD)
export SUWAYOMI_SCHEMA_VALIDATION=strict

# Warn mode: Log warnings but continue (recommended for development)
export SUWAYOMI_SCHEMA_VALIDATION=warn

# Ignore mode: Skip validation entirely
export SUWAYOMI_SCHEMA_VALIDATION=ignore
```

### Workflow Examples

#### Development Workflow

```bash
# 1. Start Suwayomi server
# 2. Fetch baseline schema
./schema-tool -server http://localhost:4567 -summary

# 3. Save baseline to git
git add schema/
git commit -m "chore: update baseline GraphQL schema"

# 4. During development, run with validation
export SUWAYOMI_SCHEMA_VALIDATION=warn
go run ./cmd/miryokusha/
```

#### CI/CD Pipeline

```bash
# In your CI pipeline:
export SUWAYOMI_SCHEMA_VALIDATION=strict

# Run tests with schema validation
go test ./... -v

# Or explicitly validate schema
./schema-tool -server $SUWAYOMI_URL -validate || exit 1
```

#### Detecting API Changes

```bash
# Update your schema periodically
./schema-tool -server http://localhost:4567

# Check what changed
git diff schema/suwayomi_schema.graphql

# Validate against old schema
git checkout HEAD~1 -- schema/suwayomi_schema.json
./schema-tool -server http://localhost:4567 -validate
```

### Understanding Validation Results

**Missing Types/Fields** ❌:
```
Type 'MangaNode' is missing from server schema
Field 'MangaNode.thumbnailUrl' is missing
```
**Action**: Update your code to handle the missing type/field or wait for server update.

**Type Mismatches** ❌:
```
MangaNode.id: expected Int!, got String!
MangaNode.chapterCount: expected Int, got Int!
```
**Action**: Update your Go structs to match the actual types.

**Deprecated Fields** ⚠️:
```
Field 'MangaNode.thumbnail' is deprecated: Use thumbnailUrl instead
```
**Action**: Update your code to use the new field name before it's removed.

### Benefits

1. **Catch Breaking Changes Early**: Know immediately when the server API changes
2. **Type Safety**: Ensure your Go structs match the GraphQL schema
3. **Documentation**: Auto-generated schema docs for your team
4. **Confidence**: Run integration tests knowing the API matches expectations
5. **Version Compatibility**: Track which Suwayomi versions your client supports

### Example Validation Output

```
✅ Schema Validation PASSED

Warnings (2):
  ⚠️  Field 'MangaNode.thumbnail' is deprecated: Use thumbnailUrl instead
  ⚠️  Mutation type is missing from server schema

Deprecated Fields (1):
  • MangaNode.thumbnail

All schema validations passed successfully!
```

### Troubleshooting

**"Unable to connect to server"**:
- Ensure Suwayomi server is running
- Check the URL is correct
- Verify firewall settings

**"Schema validation failed"**:
- Check if your Suwayomi version is compatible
- Review the validation report for specific issues
- Update your client code to match the schema
- Or update the baseline schema if server is correct

**"No expected schema file found"**:
- Run `./schema-tool` first to create the baseline
- Check the file path is correct

### Advanced Usage

**Custom Validation in Tests**:

```go
func TestSchemaCompatibility(t *testing.T) {
    client := suwayomi.NewClient(os.Getenv("SUWAYOMI_URL"))

    result, err := suwayomi.ValidateClientSchema(client)
    require.NoError(t, err)

    if !result.IsValid {
        t.Errorf("Schema validation failed:\n%s", result.Report())
    }
}
```

**Runtime Validation**:

```go
validator := suwayomi.NewRuntimeValidator(client)

// Validate in background (non-blocking)
validator.ValidateInBackground("schema/suwayomi_schema.json")

// Later, check results
if !validator.IsValid() {
    log.Println("Warning: Schema mismatch detected")
    log.Println(validator.GetValidationReport())
}
```

### See Also

- [GraphQL Introspection Specification](https://graphql.org/learn/introspection/)
- [Suwayomi Server Repository](https://github.com/Suwayomi/Suwayomi-Server)
- [Integration Tests](../../internal/suwayomi/integration_test.go)
