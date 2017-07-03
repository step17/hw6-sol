package app

import (
	"html/template"
	"net/http"
	"strings"
)

var (
	templates = template.Must(template.New("base").Funcs(funcMap).ParseGlob("pata.html"))
)

func init() {
	http.HandleFunc("/pata", handlePata)
}

func handlePata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	a := r.FormValue("a")
	b := r.FormValue("b")
	var res string
	as, bs := strings.Split(a, ""), strings.Split(b, "")
	if len(as) > 0 && len(bs) > 0 {
		for i := 0; i < len(as) || i < len(bs); i++ {
			if i < len(as) {
				res += as[i]
			}
			if i < len(bs) {
				res += bs[i]
			}
		}
	}
	err := templates.ExecuteTemplate(w, "pata.html", res)
	if err != nil {
		return err
	}
	return nil
}
