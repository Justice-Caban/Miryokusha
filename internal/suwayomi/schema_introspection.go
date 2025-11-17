package suwayomi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SchemaType represents a GraphQL type in the schema
type SchemaType struct {
	Kind          string                `json:"kind"`
	Name          string                `json:"name"`
	Description   string                `json:"description,omitempty"`
	Fields        []SchemaField         `json:"fields,omitempty"`
	InputFields   []SchemaInputValue    `json:"inputFields,omitempty"`
	Interfaces    []SchemaTypeRef       `json:"interfaces,omitempty"`
	EnumValues    []SchemaEnumValue     `json:"enumValues,omitempty"`
	PossibleTypes []SchemaTypeRef       `json:"possibleTypes,omitempty"`
}

// SchemaField represents a field in a GraphQL type
type SchemaField struct {
	Name              string             `json:"name"`
	Description       string             `json:"description,omitempty"`
	Args              []SchemaInputValue `json:"args,omitempty"`
	Type              SchemaTypeRef      `json:"type"`
	IsDeprecated      bool               `json:"isDeprecated"`
	DeprecationReason string             `json:"deprecationReason,omitempty"`
}

// SchemaInputValue represents an input value (argument or input field)
type SchemaInputValue struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	Type         SchemaTypeRef `json:"type"`
	DefaultValue string        `json:"defaultValue,omitempty"`
}

// SchemaTypeRef represents a reference to a type
type SchemaTypeRef struct {
	Kind   string         `json:"kind"`
	Name   string         `json:"name,omitempty"`
	OfType *SchemaTypeRef `json:"ofType,omitempty"`
}

// SchemaEnumValue represents an enum value
type SchemaEnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason,omitempty"`
}

// Schema represents the full GraphQL schema
type Schema struct {
	QueryType        *SchemaTypeRef `json:"queryType"`
	MutationType     *SchemaTypeRef `json:"mutationType"`
	SubscriptionType *SchemaTypeRef `json:"subscriptionType"`
	Types            []SchemaType   `json:"types"`
	Directives       []interface{}  `json:"directives"`
}

// IntrospectionResult represents the result of a schema introspection query
type IntrospectionResult struct {
	Schema       Schema    `json:"__schema"`
	FetchedAt    time.Time `json:"fetchedAt"`
	ServerURL    string    `json:"serverUrl"`
	ServerInfo   string    `json:"serverInfo,omitempty"`
}

// IntrospectSchema fetches the GraphQL schema from the server using introspection
func (c *Client) IntrospectSchema() (*IntrospectionResult, error) {
	// Standard GraphQL introspection query
	query := `
		query IntrospectionQuery {
			__schema {
				queryType { name }
				mutationType { name }
				subscriptionType { name }
				types {
					kind
					name
					description
					fields(includeDeprecated: true) {
						name
						description
						args {
							name
							description
							type {
								...TypeRef
							}
							defaultValue
						}
						type {
							...TypeRef
						}
						isDeprecated
						deprecationReason
					}
					inputFields {
						name
						description
						type {
							...TypeRef
						}
						defaultValue
					}
					interfaces {
						...TypeRef
					}
					enumValues(includeDeprecated: true) {
						name
						description
						isDeprecated
						deprecationReason
					}
					possibleTypes {
						...TypeRef
					}
				}
				directives {
					name
					description
					locations
					args {
						name
						description
						type {
							...TypeRef
						}
						defaultValue
					}
				}
			}
		}

		fragment TypeRef on __Type {
			kind
			name
			ofType {
				kind
				name
				ofType {
					kind
					name
					ofType {
						kind
						name
						ofType {
							kind
							name
							ofType {
								kind
								name
								ofType {
									kind
									name
									ofType {
										kind
										name
									}
								}
							}
						}
					}
				}
			}
		}
	`

	var response struct {
		Data struct {
			Schema Schema `json:"__schema"`
		} `json:"data"`
	}

	err := c.GraphQL.Query(query, nil, &response.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect schema: %w", err)
	}

	// Get server info if available
	serverInfo := ""
	if info, err := c.HealthCheck(); err == nil {
		serverInfo = fmt.Sprintf("%s (build: %s, revision: %s)",
			info.Version, info.BuildType, info.Revision)
	}

	result := &IntrospectionResult{
		Schema:       response.Data.Schema,
		FetchedAt:    time.Now(),
		ServerURL:    c.BaseURL,
		ServerInfo:   serverInfo,
	}

	return result, nil
}

// SaveSchemaToFile saves the introspected schema to a JSON file
func (result *IntrospectionResult) SaveSchemaToFile(filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}

// LoadSchemaFromFile loads a previously saved schema from a JSON file
func LoadSchemaFromFile(filePath string) (*IntrospectionResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var result IntrospectionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return &result, nil
}

// ToGraphQLSDL converts the schema to GraphQL Schema Definition Language format
func (result *IntrospectionResult) ToGraphQLSDL() string {
	var sdl strings.Builder

	sdl.WriteString("# GraphQL Schema\n")
	sdl.WriteString(fmt.Sprintf("# Fetched from: %s\n", result.ServerURL))
	sdl.WriteString(fmt.Sprintf("# Server: %s\n", result.ServerInfo))
	sdl.WriteString(fmt.Sprintf("# Date: %s\n", result.FetchedAt.Format(time.RFC3339)))
	sdl.WriteString("\n")

	// Write types
	for _, t := range result.Schema.Types {
		// Skip internal GraphQL types
		if strings.HasPrefix(t.Name, "__") {
			continue
		}

		switch t.Kind {
		case "OBJECT":
			result.writeObjectType(&sdl, t)
		case "INTERFACE":
			result.writeInterfaceType(&sdl, t)
		case "ENUM":
			result.writeEnumType(&sdl, t)
		case "INPUT_OBJECT":
			result.writeInputObjectType(&sdl, t)
		case "SCALAR":
			result.writeScalarType(&sdl, t)
		case "UNION":
			result.writeUnionType(&sdl, t)
		}
	}

	// Write schema definition
	sdl.WriteString("\nschema {\n")
	if result.Schema.QueryType != nil {
		sdl.WriteString(fmt.Sprintf("  query: %s\n", result.Schema.QueryType.Name))
	}
	if result.Schema.MutationType != nil {
		sdl.WriteString(fmt.Sprintf("  mutation: %s\n", result.Schema.MutationType.Name))
	}
	if result.Schema.SubscriptionType != nil {
		sdl.WriteString(fmt.Sprintf("  subscription: %s\n", result.Schema.SubscriptionType.Name))
	}
	sdl.WriteString("}\n")

	return sdl.String()
}

func (result *IntrospectionResult) writeObjectType(sdl *strings.Builder, t SchemaType) {
	if t.Description != "" {
		sdl.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", t.Description))
	}
	sdl.WriteString(fmt.Sprintf("type %s", t.Name))

	if len(t.Interfaces) > 0 {
		sdl.WriteString(" implements ")
		for i, iface := range t.Interfaces {
			if i > 0 {
				sdl.WriteString(" & ")
			}
			sdl.WriteString(iface.Name)
		}
	}

	sdl.WriteString(" {\n")
	for _, field := range t.Fields {
		result.writeField(sdl, field)
	}
	sdl.WriteString("}\n\n")
}

func (result *IntrospectionResult) writeInterfaceType(sdl *strings.Builder, t SchemaType) {
	if t.Description != "" {
		sdl.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", t.Description))
	}
	sdl.WriteString(fmt.Sprintf("interface %s {\n", t.Name))
	for _, field := range t.Fields {
		result.writeField(sdl, field)
	}
	sdl.WriteString("}\n\n")
}

func (result *IntrospectionResult) writeEnumType(sdl *strings.Builder, t SchemaType) {
	if t.Description != "" {
		sdl.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", t.Description))
	}
	sdl.WriteString(fmt.Sprintf("enum %s {\n", t.Name))
	for _, val := range t.EnumValues {
		if val.Description != "" {
			sdl.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", val.Description))
		}
		sdl.WriteString(fmt.Sprintf("  %s", val.Name))
		if val.IsDeprecated {
			sdl.WriteString(fmt.Sprintf(" @deprecated(reason: \"%s\")", val.DeprecationReason))
		}
		sdl.WriteString("\n")
	}
	sdl.WriteString("}\n\n")
}

func (result *IntrospectionResult) writeInputObjectType(sdl *strings.Builder, t SchemaType) {
	if t.Description != "" {
		sdl.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", t.Description))
	}
	sdl.WriteString(fmt.Sprintf("input %s {\n", t.Name))
	for _, field := range t.InputFields {
		if field.Description != "" {
			sdl.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", field.Description))
		}
		sdl.WriteString(fmt.Sprintf("  %s: %s", field.Name, result.typeRefToString(field.Type)))
		if field.DefaultValue != "" {
			sdl.WriteString(fmt.Sprintf(" = %s", field.DefaultValue))
		}
		sdl.WriteString("\n")
	}
	sdl.WriteString("}\n\n")
}

func (result *IntrospectionResult) writeScalarType(sdl *strings.Builder, t SchemaType) {
	if t.Description != "" {
		sdl.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", t.Description))
	}
	sdl.WriteString(fmt.Sprintf("scalar %s\n\n", t.Name))
}

func (result *IntrospectionResult) writeUnionType(sdl *strings.Builder, t SchemaType) {
	if t.Description != "" {
		sdl.WriteString(fmt.Sprintf("\"\"\"%s\"\"\"\n", t.Description))
	}
	sdl.WriteString(fmt.Sprintf("union %s = ", t.Name))
	for i, pt := range t.PossibleTypes {
		if i > 0 {
			sdl.WriteString(" | ")
		}
		sdl.WriteString(pt.Name)
	}
	sdl.WriteString("\n\n")
}

func (result *IntrospectionResult) writeField(sdl *strings.Builder, field SchemaField) {
	if field.Description != "" {
		sdl.WriteString(fmt.Sprintf("  \"\"\"%s\"\"\"\n", field.Description))
	}
	sdl.WriteString(fmt.Sprintf("  %s", field.Name))

	if len(field.Args) > 0 {
		sdl.WriteString("(")
		for i, arg := range field.Args {
			if i > 0 {
				sdl.WriteString(", ")
			}
			sdl.WriteString(fmt.Sprintf("%s: %s", arg.Name, result.typeRefToString(arg.Type)))
			if arg.DefaultValue != "" {
				sdl.WriteString(fmt.Sprintf(" = %s", arg.DefaultValue))
			}
		}
		sdl.WriteString(")")
	}

	sdl.WriteString(fmt.Sprintf(": %s", result.typeRefToString(field.Type)))

	if field.IsDeprecated {
		sdl.WriteString(fmt.Sprintf(" @deprecated(reason: \"%s\")", field.DeprecationReason))
	}

	sdl.WriteString("\n")
}

func (result *IntrospectionResult) typeRefToString(typeRef SchemaTypeRef) string {
	switch typeRef.Kind {
	case "NON_NULL":
		return result.typeRefToString(*typeRef.OfType) + "!"
	case "LIST":
		return "[" + result.typeRefToString(*typeRef.OfType) + "]"
	default:
		return typeRef.Name
	}
}

// SaveSchemaAsSDL saves the schema in GraphQL SDL format
func (result *IntrospectionResult) SaveSchemaAsSDL(filePath string) error {
	sdl := result.ToGraphQLSDL()

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, []byte(sdl), 0644); err != nil {
		return fmt.Errorf("failed to write SDL file: %w", err)
	}

	return nil
}

// GetTypeByName finds a type in the schema by name
func (result *IntrospectionResult) GetTypeByName(name string) *SchemaType {
	for i := range result.Schema.Types {
		if result.Schema.Types[i].Name == name {
			return &result.Schema.Types[i]
		}
	}
	return nil
}

// GetFieldByName finds a field in a type by name
func (t *SchemaType) GetFieldByName(name string) *SchemaField {
	for i := range t.Fields {
		if t.Fields[i].Name == name {
			return &t.Fields[i]
		}
	}
	return nil
}

// Summary returns a human-readable summary of the schema
func (result *IntrospectionResult) Summary() string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Suwayomi GraphQL Schema Summary\n"))
	summary.WriteString(fmt.Sprintf("Server: %s\n", result.ServerURL))
	summary.WriteString(fmt.Sprintf("Info: %s\n", result.ServerInfo))
	summary.WriteString(fmt.Sprintf("Fetched: %s\n\n", result.FetchedAt.Format(time.RFC3339)))

	// Count types by kind
	counts := make(map[string]int)
	for _, t := range result.Schema.Types {
		if !strings.HasPrefix(t.Name, "__") {
			counts[t.Kind]++
		}
	}

	summary.WriteString("Type Counts:\n")
	for kind, count := range counts {
		summary.WriteString(fmt.Sprintf("  %s: %d\n", kind, count))
	}

	summary.WriteString(fmt.Sprintf("\nQuery Type: %s\n", result.Schema.QueryType.Name))
	if result.Schema.MutationType != nil {
		summary.WriteString(fmt.Sprintf("Mutation Type: %s\n", result.Schema.MutationType.Name))
	}
	if result.Schema.SubscriptionType != nil {
		summary.WriteString(fmt.Sprintf("Subscription Type: %s\n", result.Schema.SubscriptionType.Name))
	}

	return summary.String()
}
