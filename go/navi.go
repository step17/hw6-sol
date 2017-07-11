package app

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

const (
	kHost = "https://fantasy-transit.appspot.com"
)

var (
	colors    = []string{"#9acd32", "#da0442", "#beaed4", "#009cd2", "#ee86a7", "#f18c43", "#9caeb7"}
	allWorlds = map[string]string{"tokyo": "東京周辺",
		"alice":    "不思議な国のアリス",
		"nausicaa": "風の谷のナウシカア",
		"lotr":     "Middle Earth (Lord of the Rings)",
		"pokemon":  "Pokemon Kanto Region",
	}
)

func init() {
	http.HandleFunc("/", handleNavi)
}

type Navi struct {
	World         string
	Worlds        map[string]string
	Network       []Line
	Lines         map[string]Line
	Adjacency     StationGraph
	LineAdjacency StationGraph
	From          string
	To            string
	Priority      Priority
	Path          Path
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
	n.LineAdjacency = LineAdjacency(n.Network)
	return nil
}

func (g StationGraph) init(x, y, line string) {
	if g[x] == nil {
		g[x] = make(map[string]map[string]bool)
	}
	if g[x][y] == nil {
		g[x][y] = make(map[string]bool)
	}
	g[x][y][line] = true
}

func Adjacency(lines []Line) StationGraph {
	g := make(StationGraph)
	for _, line := range lines {
		s := line.Stations
		if len(s) < 1 {
			continue
		}
		for i := 1; i < len(s); i++ {
			x, y := s[i], s[i-1]
			g.init(x, y, line.Name)
			g.init(y, x, line.Name)
		}
	}
	return g
}

// LineAdjacency shows what stations are connected on the same line,
// regardless of how many stops are in between.
func LineAdjacency(lines []Line) StationGraph {
	g := make(StationGraph)
	for _, line := range lines {
		s := line.Stations
		if len(s) < 2 {
			continue
		}
		for i := range s {
			for j := range s {
				g.init(s[i], s[j], line.Name)
			}
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

// BFS executes a BFS search over the given graph with from and to as
// endpoints. If line is specified, only edges from the given
// line are considered.
func (g StationGraph) BFS(ctx context.Context, from, to, line string) Path {
	if from == to {
		return nil
	}
	visited := map[string]bool{from: true}
	toVisit := []Path{{{Station: from, Line: line}}}
	for len(toVisit) > 0 {
		var path Path
		// Pop first thing from queue, keep the rest.
		path, toVisit = toVisit[0], toVisit[1:]
		last := path.Last()
		if last.Station == to {
			return path
		}

		for out, lines := range g[last.Station] {
			if line != "" && !lines[line] {
				continue
			}
			if visited[out] {
				continue
			}
			toVisit = append(toVisit, path.Grow(out, lines))
			visited[out] = true
		}
	}
	// No path found.
	return nil
}

func (n Navi) Route(ctx context.Context, from, to string) Path {
	switch n.Priority {
	case FewerTransfers:
		landmarks := n.LineAdjacency.BFS(ctx, from, to, "")
		if len(landmarks) < 2 {
			return nil
		}
		path := Path{{Station: from}}
		// Find individual hops between landmarks.
		for i := 1; i < len(landmarks); i++ {
			subPath := n.Adjacency.BFS(ctx, landmarks[i-1].Station, landmarks[i].Station, landmarks[i].Line)
			path = append(path, subPath[1:]...)
		}
		// Add line to first hop just to look nicer.
		path[0].Line = path[1].Line
		return path
	default:
		return n.Adjacency.BFS(ctx, from, to, "")
	}
}

// Priority is what to prioritize when making a route
type Priority int

const (
	FewerStations Priority = iota
	FewerTransfers
)

func (p Priority) String() string {
	switch p {
	case FewerStations:
		return "駅数が少ない"
	case FewerTransfers:
		return "乗り換え少ない"
	}
	return "Unknown"
}

func AsPriority(p string) Priority {
	switch p {
	case "駅数が少ない":
		return FewerStations
	case "乗り換え少ない":
		return FewerTransfers
	default:
		return FewerStations
	}
}

var (
	tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"LineColor": func(s string) string { return "" },
		"WorldLink": func(s string) string { return "" },
	}).ParseGlob("*.html"))
)

func LoadNavi(r *http.Request) (context.Context, Navi) {
	ctx := appengine.NewContext(r)
	hostname := appengine.DefaultVersionHostname(ctx)
	n := Navi{
		Worlds:   allWorlds,
		World:    r.FormValue("world"),
		From:     r.FormValue("from"),
		To:       r.FormValue("to"),
		Priority: AsPriority(r.FormValue("priority")),
	}
	if n.World == "" {
		hostParts := strings.Split(hostname, ".")
		log.Infof(ctx, "host %v -> hostparts: %v", hostname, hostParts)
		if _, exists := allWorlds[hostParts[0]]; exists {
			n.World = hostParts[0]
		}
	}
	if err := n.LoadNet(ctx); err != nil {
		panic(err)
	}
	tmpl.Funcs(template.FuncMap{
		"LineColor": func(s string) string {
			return n.Lines[s].Color
		},
		"WorldLink": func(w string) string {
			u := *r.URL
			hostParts := strings.Split(hostname, ".")
			if _, exists := allWorlds[hostParts[0]]; exists {
				hostParts = hostParts[1:]
			}
			hostParts = append([]string{w}, hostParts...)
			u.Host = strings.Join(hostParts, ".")
			u.RawQuery = ""
			return u.String()
		}},
	)
	return ctx, n
}

func handleNavi(w http.ResponseWriter, r *http.Request) {
	ctx, n := LoadNavi(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if n.Adjacency.Exists(n.From) && n.Adjacency.Exists(n.To) {
		n.Path = n.Route(ctx, n.From, n.To)
	}
	err := tmpl.ExecuteTemplate(w, "navi.html", n)
	if err != nil {
		panic(err)
	}
}
