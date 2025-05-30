package main

import (
	"fmt"
	"strings"
)

func main() {
	d:= []string{}
	g:="HEAD"
	var sb strings.Builder

	sb.WriteString("jkh'lkhl")
	sb.WriteString(fmt.Sprintf("\nhklhlk %s", g))
	sb.WriteString("\n\n")
	fmt.Println(sb.String())

	for _, v := range d {
		sb.WriteString(fmt.Sprintf("\n%s", v))
	}
	
	
}