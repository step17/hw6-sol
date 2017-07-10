package app

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

const (
	kHost = "https://fantasy-transit.appspot.com"
)

var (
	colors = []string{"#386cb0", "#7fc97f", "#beaed4", "#bf5b17", "#f0027f", "#fdc086", "#ff83fa"}
)

func init() {
	http.HandleFunc("/", handleNavi)
}

type Navi struct {
	World     string
	Network   []Line
	Adjacency StationGraph
	From      string
	To        string
	Path      []string
}

type Line struct {
	Name     string
	Stations []string
	Color    string
}

// StationGraph is a map of StationX->StationY combinations showing
// whether there's an from StationX to StationY.
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
	// Add line colors for fun
	for li := range n.Network {
		if li < len(colors) {
			n.Network[li].Color = colors[li]
		}
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
		for i := 1; i < len(s); i++ {
			x, y := s[i], s[i-1]
			init(x)
			init(y)
			g[x][y] = true
			g[y][x] = true
		}
	}
	return g
}

// Exists returns true iff the given station exists in the graph.
func (g StationGraph) Exists(station string) bool {
	_, ok := g[station]
	return ok
}

// Path is a convenience type for representing a path though a series
// of stations.
type Path []string

// Last returns the last step in a path. Panics if p is empty.
func (p Path) Last() string {
	return p[len(p)-1]
}

// Grow makes a new Path that includes the given path plus a new
// station.
func (p Path) Grow(station string) Path {
	n := make(Path, len(p), len(p)+1)
	copy(n, p)
	n = append(n, station)
	return n
}

func (n Navi) Route(ctx context.Context, from, to string) Path {
	visited := map[string]bool{from: true}
	toVisit := []Path{{from}}
	for len(toVisit) > 0 {
		var path Path
		// Pop first thing from queue, keep the rest.
		path, toVisit = toVisit[0], toVisit[1:]
		last := path.Last()
		if last == to {
			return path
		}
		for out := range n.Adjacency[last] {
			if visited[out] {
				continue
			}
			add := path.Grow(out)
			toVisit = append(toVisit, add)
			visited[out] = true
		}
	}
	// No path found.
	return nil
}

var (
	tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"Skip1": func(s []string) []string { return s[1:] },
	}).ParseGlob("*.html"))
)

func handleNavi(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	n := Navi{
		World: r.FormValue("world"),
		From:  r.FormValue("from"),
		To:    r.FormValue("to"),
	}
	if err := n.LoadNet(ctx); err != nil {
		panic(err)
	}

	if n.Adjacency.Exists(n.From) && n.Adjacency.Exists(n.To) {
		n.Path = n.Route(ctx, n.From, n.To)
	}
	err := tmpl.ExecuteTemplate(w, "navi.html", n)
	if err != nil {
		panic(err)
	}
}
