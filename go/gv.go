// This gv.go file contains some extra debug/visualization stuff
// unrelated to the actual train routing logic.
package app

import (
	"fmt"
	"io"
	"net/http"
)

func init() {
	http.HandleFunc("/gv", handleGV)
}

func (n Navi) GV(w io.Writer, g StationGraph, penWidth int) {
	fmt.Fprintln(w, `graph g {`)
	fmt.Fprintln(w, `  graph [overlap=scale]`)
	done := make(map[string]bool)
	keyFn := func(x, y string) string {
		if x < y {
			y, x = x, y
		}
		return x + ":" + y
	}
	for x, ym := range g {
		for y, lines := range ym {
			key := keyFn(x, y)
			if done[key] || x == y {
				continue
			}
			for line := range lines {
				fmt.Fprintf(w, `  "%s" -- "%s" [color="%s" penwidth=%d]`, x, y, n.Lines[line].Color, penWidth)
				fmt.Fprintf(w, "\n")
			}
			done[key] = true
		}
	}
	fmt.Fprintln(w, "}")
}

func handleGV(w http.ResponseWriter, r *http.Request) {
	_, n := LoadNavi(r)
	w.Header().Set("Content-Type", "text/gv; charset=utf-8")
	switch r.FormValue("adj") {
	case "lines":
		n.GV(w, n.LineAdjacency, 1)
	default:
		n.GV(w, n.Adjacency, 5)
	}
}
