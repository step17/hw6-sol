package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestPata(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		{"パトカー", "タクシー", "パタトクカシーー"},
		{"hamster", "lobster", "hlaombsstteerr"},
	}
	for _, test := range tests {
		req, _ := http.NewRequest("GET", "/pata", nil)
		req.Form = make(url.Values)
		req.Form["a"] = []string{test.a}
		req.Form["b"] = []string{test.b}
		w := httptest.NewRecorder()
		handlePata(w, req)
		if !strings.Contains(w.Body.String(), test.want) {
			t.Errorf("/pata with a=%v b=%v got: %v wanted %v", test.a, test.b, w.Body.String(), test.want)
		}
	}
}
