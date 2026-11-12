package jsonc

import (
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

func TestIssue6(t *testing.T) {
	src := "{// c\r\"a\": 1}"
	exp := "{    \r\"a\": 1}"
	got := string(ToJSON([]byte(src)))
	if got != exp {
		t.Fatalf("expected '%s', got '%s'", exp, got)
	}

	src = "{// c\r\n\"a\": 1}"
	exp = "{    \r\n\"a\": 1}"
	got = string(ToJSON([]byte(src)))
	if got != exp {
		t.Fatalf("expected '%s', got '%s'", exp, got)
	}
	src = "{// c\n\r\"a\": 1}"
	exp = "{    \n\r\"a\": 1}"
	got = string(ToJSON([]byte(src)))
	if got != exp {
		t.Fatalf("expected '%s', got '%s'", exp, got)
	}

}
