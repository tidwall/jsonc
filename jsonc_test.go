package jsonc

import (
	"strings"
	"testing"
)

func testToJSON(t *testing.T, json, expect string) {
	t.Helper()
	if len(json) != len(expect) {
		t.Fatal()
	}
	out := string(ToJSON([]byte(json)))
	if out != expect {
		t.Fatalf("expected '%s', got '%s'", expect, out)
	}
	out = string(ToJSONInPlace([]byte(json)))
	if out != expect {
		t.Fatalf("expected '%s', got '%s'", expect, out)
	}
}

func TestToJSON(t *testing.T) {
	testToJSON(t, `
  {  //	hello
    "c": 3,"b":3, // jello
    /* SOME
       LIKE
       IT
       HAUT */
    "d\\\"\"e": [ 1, /* 2 */ 3, 4, ],
  }`, `
  {    	     
    "c": 3,"b":3,         
           
           
         
              
    "d\\\"\"e": [ 1,         3, 4  ] 
  }`)
}

func TestIssue3(t *testing.T) {
	testToJSON(t,
		`{"a":1}/* unclosed
  asdasdf  asdfsadf */ /* asdf`,
		`{"a":1}           
                       /*     `,
	)
	testToJSON(t,
		`{"a":1}/* unclosed
  asdasdf  asdfsadf */ /* asdf*/`,
		`{"a":1}           
                                `,
	)
}

func TestLineCommentLineEndings(t *testing.T) {
	for _, ending := range []struct {
		name  string
		value string
	}{
		{"CR", "\r"},
		{"CRLF", "\r\n"},
		{"LF", "\n"},
	} {
		t.Run(ending.name, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				input  string
				expect string
			}{
				{"before object", "// hi\n{\"a\":1}", "     \n{\"a\":1}"},
				{"between members", "{\"a\":1, // hi\n\"b\":2}", "{\"a\":1,      \n\"b\":2}"},
				{"between elements", "[1,// hi\n2,]", "[1,     \n2 ]"},
				{"before closing brace", "{\"a\":1,// hi\n}", "{\"a\":1      \n}"},
				{"consecutive comments", "// first\n// second\n{\"a\":1}", "        \n         \n{\"a\":1}"},
				{"at end of input", "{\"a\":1}// hi\n", "{\"a\":1}     \n"},
				{"quoted markers", "{\"s\":\"// hi\\r\",// hi\n\"b\":2}", "{\"s\":\"// hi\\r\",     \n\"b\":2}"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					testToJSON(t,
						strings.ReplaceAll(tc.input, "\n", ending.value),
						strings.ReplaceAll(tc.expect, "\n", ending.value),
					)
				})
			}
		})
	}
}
