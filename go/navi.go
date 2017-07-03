package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

const (
	kHost = "https://fantasy-transit.appspot.com"
)

func init() {
	http.HandleFunc("/", handleNavi)
}

type Navi struct {
	World     string
	Network   []Line
	Adjacency StationGraph
}

type Line struct {
	Name     string
	Stations []string
}

type StationGraph map[string]map[string]bool

func (n *Navi) LoadNet(ctx context.Context) error {
	req, err := http.NewRequest("GET", kHost, nil)
	if err != nil {
		return err
	}
	req.URL.Path = "/net"
	q := url.Values{}
	q.Set("world", n.World)
	q.Set("format", "json")
	req.URL.RawQuery = q.Encode()
	client := urlfetch.Client(ctx)
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	dec := json.NewDecoder(resp.Body)
	if err = dec.Decode(&n.Network); err != nil {
		return err
	}
	n.Adjacency = Adjacency(n.Network)
	return nil
}

func Adjacency(lines []Line) StationGraph {
	g := make(StationGraph)
	init := func(x string) {
		if g[x] == nil {
			g[x] = make(map[string]bool)
		}
	}
	for _, line := range lines {
		s := line.Stations
		if len(s) < 1 {
			continue
		}
		init(s[0])
		for i := 1; i < len(s); i++ {
			x, y := s[i], s[i-1]
			init(x)
			g[x][y] = true
			g[y][x] = true
		}
	}
	return g
}

type Page struct {
	World string
}

func handleNavi(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	n := Navi{World: r.FormValue("world")}
	if err := n.LoadNet(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprintf(w, "%#v", n)
}
