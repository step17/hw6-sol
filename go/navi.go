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
	colors = []string{"#9acd32", "#da0442", "#beaed4", "#009cd2", "#ee86a7", "#f18c43", "#9caeb7"}
)

func init() {
	http.HandleFunc("/", handleNavi)
}

type Navi struct {
	World     string
	Network   []Line
	Lines     map[string]Line
	Adjacency StationGraph
	From      string
	To        string
	Path      Path
}

type Line struct {
	Name     string
	Stations []string
	Color    string
}

// StationGraph is a map of StationX->StationY->LineZ combinations
// showing whether there's an edge representing a link StationX to
// StationY on LineZ.
type StationGraph map[string]map[string]map[string]bool

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
	n.Lines = make(map[string]Line)
	// Add line colors for fun
	for li := range n.Network {
		line := &n.Network[li]
		if li < len(colors) {
			line.Color = colors[li]
		}
		n.Lines[line.Name] = *line
	}
	n.Adjacency = Adjacency(n.Network)
	return nil
}

func Adjacency(lines []Line) StationGraph {
	g := make(StationGraph)
	for _, line := range lines {
		init := func(x, y string) {
			if g[x] == nil {
				g[x] = make(map[string]map[string]bool)
			}
			if g[x][y] == nil {
				g[x][y] = make(map[string]bool)
			}
			g[x][y][line.Name] = true
		}
		s := line.Stations
		if len(s) < 1 {
			continue
		}
		for i := 1; i < len(s); i++ {
			x, y := s[i], s[i-1]
			init(x, y)
			init(y, x)
		}
	}
	return g
}

// Exists returns true iff the given station exists in the graph.
func (g StationGraph) Exists(station string) bool {
	_, ok := g[station]
	return ok
}

// Path represents a path though a series of stations.
type Path []Hop

// A hop is one point in a path.
type Hop struct {
	Station string
	// Line represents the incoming line, if set.
	Line string
}

// Last returns the last step in a path. Panics if p is empty.
func (p Path) Last() *Hop {
	return &p[len(p)-1]
}

// Grow makes a new Path that includes the given path plus a new
// station.
func (p Path) Grow(station string, lines map[string]bool) Path {
	n := make(Path, len(p), len(p)+1)
	copy(n, p)
	hop := Hop{Station: station}
	last := n.Last()
	if lines[last.Line] { // stay on same line if possible
		hop.Line = last.Line
	} else { // otherwise pick one arbitrarily.
		for line := range lines {
			hop.Line = line
			break
		}
	}
	if last.Line == "" {
		last.Line = hop.Line
	}
	n = append(n, hop)
	return n
}

func (n Navi) Route(ctx context.Context, from, to string) Path {
	if from == to {
		return nil
	}
	visited := map[string]bool{from: true}
	toVisit := []Path{{{Station: from}}}
	for len(toVisit) > 0 {
		var path Path
		// Pop first thing from queue, keep the rest.
		path, toVisit = toVisit[0], toVisit[1:]
		last := path.Last()
		if last.Station == to {
			return path
		}
		for out, lines := range n.Adjacency[last.Station] {
			if visited[out] {
				continue
			}
			add := path.Grow(out, lines)
			toVisit = append(toVisit, add)
			visited[out] = true
		}
	}
	// No path found.
	return nil
}

var (
	tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"LineColor": func(s string) string { return "" },
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
	err := tmpl.Funcs(template.FuncMap{
		"LineColor": func(s string) string {
			return n.Lines[s].Color
		}}).ExecuteTemplate(w, "navi.html", n)
	if err != nil {
		panic(err)
	}
}
