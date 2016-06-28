package app

import (
	"fmt"
	"net/http"
)

func init() {
	http.HandleFunc("/", handlePata)
}

func handlePata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<body>
<i>Hello world!</i> in Japanese is <i>こんにちは世界!</i>
</body>`)
}
