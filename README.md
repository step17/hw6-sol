# HW6 example solution

This is an example solution for STEP HW6, implemented in Go.

Part 1:

- パタトクカシーー is at http://fantasy-transit-example.appspot.com/pata

Part 2:

- http://fantasy-transit-example.appspot.com/

Ther's two choices for priorities in routing: least transfers, or least stations in the path.

To compute the "least stations" path, it's simply computed
via [BFS](https://en.wikipedia.org/wiki/Breadth-first_search) over the train
station network.

To compute the "least transfers" path, the train station network graph is
transformed into an atlernate graph where all of the stations on the same line
share an edge. On this graph, a
shortest [BFS](https://en.wikipedia.org/wiki/Breadth-first_search) path finds
the stations where transfers should happen. Then between each of these "landmark
stations", BFS is used again over the single line used between those "landmark
stations" to enumerate a stations between them.

For example this adjacency graph for Tokyo:

![Tokyo adjacency](tokyo-adj.png)

And the corresponding *line* adjacency graph where stations on the same line are connected:

![Tokyo adjacency](tokyo-line-adj.png)
