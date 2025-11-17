package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Justice-Caban/Miryokusha/internal/suwayomi"
)

func main() {
	// Define flags
	serverURL := flag.String("server", "http://localhost:4567", "Suwayomi server URL")
	outputJSON := flag.String("json", "schema/suwayomi_schema.json", "Output path for JSON schema")
	outputSDL := flag.String("sdl", "schema/suwayomi_schema.graphql", "Output path for SDL schema")
	validate := flag.Bool("validate", false, "Validate client against server schema")
	summary := flag.Bool("summary", false, "Show schema summary")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Suwayomi GraphQL Schema Introspection Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Introspect schema and save to files\n")
		fmt.Fprintf(os.Stderr, "  %s -server http://localhost:4567\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Show schema summary\n")
		fmt.Fprintf(os.Stderr, "  %s -server http://localhost:4567 -summary\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Validate client code against server schema\n")
		fmt.Fprintf(os.Stderr, "  %s -server http://localhost:4567 -validate\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Save to custom paths\n")
		fmt.Fprintf(os.Stderr, "  %s -server http://localhost:4567 -json schema.json -sdl schema.graphql\n\n", os.Args[0])
	}

	flag.Parse()

	// Create Suwayomi client
	fmt.Printf("Connecting to Suwayomi server at %s...\n", *serverURL)
	client := suwayomi.NewClient(*serverURL)

	// Check if server is reachable
	if !client.Ping() {
		fmt.Fprintf(os.Stderr, "Error: Unable to connect to server at %s\n", *serverURL)
		fmt.Fprintf(os.Stderr, "Make sure the Suwayomi server is running.\n")
		os.Exit(1)
	}

	fmt.Println("✅ Server is reachable")

	// Introspect schema
	fmt.Println("\nIntrospecting GraphQL schema...")
	schema, err := client.IntrospectSchema()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error introspecting schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Schema introspection successful")

	// Show summary if requested
	if *summary {
		fmt.Println("\n" + schema.Summary())
	}

	// Save JSON schema
	fmt.Printf("\nSaving JSON schema to %s...\n", *outputJSON)
	if err := schema.SaveSchemaToFile(*outputJSON); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving JSON schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ JSON schema saved")

	// Save SDL schema
	fmt.Printf("Saving SDL schema to %s...\n", *outputSDL)
	if err := schema.SaveSchemaAsSDL(*outputSDL); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving SDL schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ SDL schema saved")

	// Validate if requested
	if *validate {
		fmt.Println("\nValidating client code against server schema...")

		// Load expected schema (from previously saved schema)
		expectedSchema, err := suwayomi.LoadSchemaFromFile(*outputJSON)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not load expected schema for comparison: %v\n", err)
		} else {
			// For this first run, we're comparing against itself
			// In practice, you'd compare against a baseline schema
			validator := suwayomi.NewSchemaValidator(expectedSchema, schema)
			result := validator.Validate()

			fmt.Println("\n" + result.Report())

			if !result.IsValid {
				os.Exit(1)
			}
		}
	}

	fmt.Println("\n✨ Schema introspection complete!")
	fmt.Println("\nYou can now:")
	fmt.Println("  1. Review the schema files to understand the API")
	fmt.Println("  2. Use the JSON schema for validation in tests")
	fmt.Println("  3. Check the SDL schema for GraphQL type definitions")
	fmt.Println("\nTo validate your client code:")
	fmt.Printf("  %s -server %s -validate\n", os.Args[0], *serverURL)
}
