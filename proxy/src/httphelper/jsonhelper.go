package httphelper

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func PrintJson(b []byte) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "\t")
	if err != nil {
		fmt.Printf("\nError parsing JSON\n%s\n", b)
	} else {
		//out.WriteTo(os.Stdout)
		fmt.Printf("\n%s\n", out.Bytes())
	}
}

func PrettyJson(b []byte) string {
	var out bytes.Buffer
	var s string
	err := json.Indent(&out, b, "", "\t")
	if err != nil {
		s = fmt.Sprintf("%s\n", b)
	} else {
		//out.WriteTo(os.Stdout)
		s = fmt.Sprintf("%s", out.Bytes())
	}
	return s
}
