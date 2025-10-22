package jsonc

import "testing"

func TestToJSON(t *testing.T) {
	json := `
  {  //	hello
    "c": 3,"b":3, // jello
    /* SOME
       LIKE
       IT
       HAUT */
    "d\\\"\"e": [ 1, /* 2 */ 3, 4, ],
  }`
	expect := `
  {    	     
    "c": 3,"b":3,         
           
           
         
              
    "d\\\"\"e": [ 1,         3, 4  ] 
  }`
	out := string(ToJSON([]byte(json)))
	if out != expect {
		t.Fatalf("expected '%s', got '%s'", expect, out)
	}
	out = string(ToJSONInPlace([]byte(json)))
	if out != expect {
		t.Fatalf("expected '%s', got '%s'", expect, out)
	}
}

func TestParseAndModify(t *testing.T) {
	jsonc := `{
  // Configuration for dev environment
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "dev",
  },
  "features": [
    "auth",
    "logging", // Important feature
  ]
}`

	// Parse the JSONC
	doc, err := Parse([]byte(jsonc))
	if err != nil {
		t.Fatalf("failed to parse JSONC: %v", err)
	}

	// Test getting values
	host, err := doc.Get("database.host")
	if err != nil {
		t.Fatalf("failed to get database.host: %v", err)
	}
	if host != "localhost" {
		t.Fatalf("expected 'localhost', got '%v'", host)
	}

	// Test setting values
	err = doc.Set("database.password", "secret123")
	if err != nil {
		t.Fatalf("failed to set database.password: %v", err)
	}

	// Test setting nested values
	err = doc.Set("cache.redis.host", "redis.example.com")
	if err != nil {
		t.Fatalf("failed to set cache.redis.host: %v", err)
	}

	// Test array access
	feature0, err := doc.Get("features.0")
	if err != nil {
		t.Fatalf("failed to get features.0: %v", err)
	}
	if feature0 != "auth" {
		t.Fatalf("expected 'auth', got '%v'", feature0)
	}

	// Convert back to JSONC
	result, err := doc.ToJSONC()
	if err != nil {
		t.Fatalf("failed to convert to JSONC: %v", err)
	}

	// The result should be valid and parseable
	doc2, err := Parse(result)
	if err != nil {
		t.Fatalf("failed to parse generated JSONC: %v", err)
	}

	// Verify the new values exist
	password, err := doc2.Get("database.password")
	if err != nil {
		t.Fatalf("failed to get database.password from generated JSONC: %v", err)
	}
	if password != "secret123" {
		t.Fatalf("expected 'secret123', got '%v'", password)
	}

	redisHost, err := doc2.Get("cache.redis.host")
	if err != nil {
		t.Fatalf("failed to get cache.redis.host from generated JSONC: %v", err)
	}
	if redisHost != "redis.example.com" {
		t.Fatalf("expected 'redis.example.com', got '%v'", redisHost)
	}
}

func TestParseSimpleObject(t *testing.T) {
	jsonc := `{"name": "test", "value": 42}`

	doc, err := Parse([]byte(jsonc))
	if err != nil {
		t.Fatalf("failed to parse simple object: %v", err)
	}

	if doc.Root.Type != NodeTypeObject {
		t.Fatalf("expected object, got %d", doc.Root.Type)
	}

	if len(doc.Root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(doc.Root.Children))
	}
}

func TestParseArray(t *testing.T) {
	jsonc := `[1, "two", true, null]`

	doc, err := Parse([]byte(jsonc))
	if err != nil {
		t.Fatalf("failed to parse array: %v", err)
	}

	if doc.Root.Type != NodeTypeArray {
		t.Fatalf("expected array, got %d", doc.Root.Type)
	}

	if len(doc.Root.Children) != 4 {
		t.Fatalf("expected 4 children, got %d", len(doc.Root.Children))
	}

	// Check types
	if doc.Root.Children[0].Type != NodeTypeNumber {
		t.Fatalf("expected number at index 0")
	}
	if doc.Root.Children[1].Type != NodeTypeString {
		t.Fatalf("expected string at index 1")
	}
	if doc.Root.Children[2].Type != NodeTypeBool {
		t.Fatalf("expected bool at index 2")
	}
	if doc.Root.Children[3].Type != NodeTypeNull {
		t.Fatalf("expected null at index 3")
	}
}

func TestTrailingCommas(t *testing.T) {
	jsonc := `{
  "array": [1, 2, 3,],
  "object": {
    "key": "value",
  },
}`

	doc, err := Parse([]byte(jsonc))
	if err != nil {
		t.Fatalf("failed to parse JSONC with trailing commas: %v", err)
	}

	// Check that trailing commas are preserved
	arrayNode := doc.Root.Children[0] // "array" property node (which is the array itself)
	if arrayNode.Type != NodeTypeArray {
		t.Fatalf("expected array type, got %d", arrayNode.Type)
	}
	if !arrayNode.HasTrailingComma { // Check the array itself has trailing comma
		t.Fatalf("expected trailing comma in array to be preserved")
	}

	objectNode := doc.Root.Children[1] // "object" property node (which is the object itself)
	if objectNode.Type != NodeTypeObject {
		t.Fatalf("expected object type, got %d", objectNode.Type)
	}
	if !objectNode.HasTrailingComma { // Check the object itself has trailing comma
		t.Fatalf("expected trailing comma in nested object to be preserved")
	}
}

func TestDelete(t *testing.T) {
	jsonc := `{
  "keep": "this",
  "delete": "this",
  "array": [1, 2, 3]
}`

	doc, err := Parse([]byte(jsonc))
	if err != nil {
		t.Fatalf("failed to parse JSONC: %v", err)
	}

	// Delete a property
	err = doc.Delete("delete")
	if err != nil {
		t.Fatalf("failed to delete property: %v", err)
	}

	// Verify it's gone
	_, err = doc.Get("delete")
	if err == nil {
		t.Fatalf("expected error when getting deleted property")
	}

	// Delete array element
	err = doc.Delete("array.1")
	if err != nil {
		t.Fatalf("failed to delete array element: %v", err)
	}

	// The array should now have 2 elements instead of 3
	// Find the array node
	var arrayNode *Node
	for _, child := range doc.Root.Children {
		if child.Key == "array" {
			arrayNode = child
			break
		}
	}
	if arrayNode == nil {
		t.Fatalf("could not find array node")
	}

	if len(arrayNode.Children) != 2 {
		t.Fatalf("expected array to have 2 elements after deletion, got %d", len(arrayNode.Children))
	}
}

func TestRoundTrip(t *testing.T) {
	original := `{
  "string": "hello world",
  "number": 42.5,
  "bool": true,
  "null": null,
  "array": [1, 2, 3],
  "object": {
    "nested": "value"
  }
}`

	// Parse
	doc, err := Parse([]byte(original))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Convert back
	result, err := doc.ToJSONC()
	if err != nil {
		t.Fatalf("failed to convert back: %v", err)
	}

	// Parse again
	doc2, err := Parse(result)
	if err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	// Verify values are preserved
	str, _ := doc2.Get("string")
	if str != "hello world" {
		t.Fatalf("string value not preserved")
	}

	num, _ := doc2.Get("number")
	if num != "42.5" { // Numbers are stored as strings in our simple implementation
		t.Fatalf("number value not preserved")
	}

	b, _ := doc2.Get("bool")
	if b != true {
		t.Fatalf("bool value not preserved")
	}

	nested, _ := doc2.Get("object.nested")
	if nested != "value" {
		t.Fatalf("nested value not preserved")
	}
}
