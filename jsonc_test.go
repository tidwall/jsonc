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
    "d": [ 1, /* 2 */ 3, 4, ],
  }`
	expect := `
  {    	     
    "c": 3,"b":3,         
           
           
         
              
    "d": [ 1,         3, 4  ] 
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
