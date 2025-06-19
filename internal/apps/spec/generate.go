package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/digitalocean/godo"
	"github.com/invopop/jsonschema"
)

//go:generate go run .

func main() {
	reflect := jsonschema.Reflector{
		BaseSchemaID:               "",
		Anonymous:                  true,
		AssignAnchor:               false,
		AllowAdditionalProperties:  true,
		RequiredFromJSONSchemaTags: true,
		DoNotReference:             true,
		ExpandedStruct:             true,
		FieldNameTag:               "",
	}

	err := reflect.AddGoComments("github.com/digitalocean/godo", "./")
	if err != nil {
		panic(fmt.Errorf("failed to add Go comments: %w", err))
	}

	createSchema, err := reflect.Reflect(&godo.AppCreateRequest{}).MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("failed to marshal app create schema: %w", err))
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, createSchema, "", "  "); err != nil {
		panic(fmt.Errorf("failed to indent JSON: %w", err))
	}

	// now write the schema to a file
	err = os.WriteFile("./app-create-schema.json", prettyJSON.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}

	fmt.Println("Schema successfully written to app_create_schema.json")

	// Generate schema for AppUpdateRequest
	updateSchema, err := reflect.Reflect(&godo.AppUpdateRequest{}).MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("failed to marshal app update schema: %w", err))
	}

	// Prettify the JSON
	var prettyUpdateJSON bytes.Buffer
	if err := json.Indent(&prettyUpdateJSON, updateSchema, "", "  "); err != nil {
		panic(fmt.Errorf("failed to indent JSON: %w", err))
	}

	// Write the schema to a file
	err = os.WriteFile("./app-update-schema.json", prettyUpdateJSON.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write schema to file: %w", err))
	}

	fmt.Println("Update schema successfully written to app_update_schema.json")
}
