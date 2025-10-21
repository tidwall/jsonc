# jsonc

[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg?style=flat-square)](https://pkg.go.dev/github.com/tidwall/jsonc) 

jsonc is a Go package that converts the jsonc format to standard json.

The jsonc format is like standard json but allows for comments and trailing
commas, such as:

```js
{

  /* Dev Machine */
  "dbInfo": {
    "host": "localhost",
    "port": 5432,          
    "username": "josh",
    "password": "pass123", // please use a hashed password
  },

  /* Only SMTP Allowed */
  "emailInfo": {
    "email": "josh@example.com", // use full email address
    "password": "pass123",
    "smtp": "smpt.example.com",
  }

}
```

There are several functions available for working with JSONC:

- `jsonc.ToJSON` - Converts JSONC to standard JSON (strips comments and trailing commas)
- `jsonc.Parse` - Parses JSONC into a document structure that preserves formatting
- Document methods for reading and modifying JSONC while preserving formatting:
  - `Get(path)` - Get a value at a given path
  - `Set(path, value)` - Set a value at a given path
  - `Delete(path)` - Delete a value at a given path
  - `ToJSONC()` - Convert back to JSONC format with preserved formatting

The `ToJSON` function ensures the resulting JSON will always be the same length as the input and it will
include all of the same line breaks at matching offsets. This is to ensure
the result can be later processed by a external parser and that that
parser will report messages or errors with the correct offsets.

## Getting Started

### Installing

To start using jsonc, install Go and run `go get`:

```sh
$ go get -u github.com/tidwall/jsonc
```

This will retrieve the library.

### Examples

#### Converting JSONC to JSON

The following example uses a JSONC document that has comments and trailing
commas and converts it just prior to unmarshalling with the standard Go
JSON library.

```go
data := `
{
  /* Dev Machine */
  "dbInfo": {
    "host": "localhost",
    "port": 5432,          // use full email address
    "username": "josh",
    "password": "pass123", // use a hashed password
  },
  /* Only SMTP Allowed */
  "emailInfo": {
    "email": "josh@example.com",
    "password": "pass123",
    "smtp": "smpt.example.com",
  }
}
`

err := json.Unmarshal(jsonc.ToJSON(data), &config)
```

#### Parsing and Modifying JSONC

The following example shows how to parse JSONC, modify values, and write back to JSONC format while preserving comments and formatting:

```go
import "github.com/tidwall/jsonc"

// Original JSONC with comments and trailing commas
data := `{
  // Database configuration
  "database": {
    "host": "localhost",
    "port": 5432,
    "user": "dev",
  },
  "features": [
    "auth",
    "logging", // Important for debugging
  ]
}`

// Parse JSONC
doc, err := jsonc.Parse([]byte(data))
if err != nil {
    panic(err)
}

// Read values
host, err := doc.Get("database.host")
if err != nil {
    panic(err)
}
fmt.Printf("Database host: %s\n", host)

// Modify existing values
err = doc.Set("database.port", 3306)
if err != nil {
    panic(err)
}

// Add new values
err = doc.Set("database.password", "secret123")
if err != nil {
    panic(err)
}

// Add nested structures
err = doc.Set("cache.redis.host", "redis.example.com")
if err != nil {
    panic(err)
}

// Delete values
err = doc.Delete("features.1") // Remove "logging" feature
if err != nil {
    panic(err)
}

// Convert back to JSONC format (preserves comments and trailing commas)
result, err := doc.ToJSONC()
if err != nil {
    panic(err)
}

fmt.Printf("Modified JSONC:\n%s\n", result)
```

The modified JSONC output will preserve comments, trailing commas, and formatting while incorporating your changes.

#### Path Syntax

Paths use dot notation to access nested values:

- `"key"` - Access a property in an object
- `"parent.child"` - Access nested properties
- `"array.0"` - Access array elements by index
- `"parent.array.1.property"` - Complex nested access

Examples:
```go
// Object access
value, _ := doc.Get("database.host")

// Array access  
firstFeature, _ := doc.Get("features.0")

// Nested array/object access
nestedValue, _ := doc.Get("users.0.profile.name")

// Setting creates intermediate objects/arrays as needed
doc.Set("new.nested.value", "hello")
```

### Performance

It's fast and can convert GB/s of jsonc to json.

## Contact

Josh Baker [@tidwall](http://twitter.com/tidwall)

## License

jsonc source code is available under the MIT [License](/LICENSE).
