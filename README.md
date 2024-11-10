# Tile: Data-Oriented 2D Grid Engine

<p align="center">
    <img width="340" height="152" src="./fixtures/logo.png">
    <br>
    <img src="https://img.shields.io/github/go-mod/go-version/kelindar/tile" alt="Go Version">
    <a href="https://pkg.go.dev/github.com/kelindar/tile"><img src="https://pkg.go.dev/badge/github.com/kelindar/tile" alt="PkgGoDev"></a>
    <a href="https://goreportcard.com/report/github.com/kelindar/tile"><img src="https://goreportcard.com/badge/github.com/kelindar/tile" alt="Go Report Card"></a>
    <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
    <a href="https://coveralls.io/github/kelindar/tile"><img src="https://coveralls.io/repos/github/kelindar/tile/badge.svg" alt="Coverage"></a>
</p>

This repository contains a 2D tile map engine which is built with data and cache friendly ways. My main goal here is to provide a simple, high performance library to handle large scale tile maps in games.

- **Compact**. Each tile value is 4 bytes long and each grid page is 64-bytes long, which means a grid of 3000x3000 should take around 64MB of memory.
- **Thread-safe**. The grid is thread-safe and can be updated through provided update function. This allows multiple goroutines to read/write to the grid concurrently without any contentions. There is a spinlock per tile page protecting tile access.
- **Views & observers**. When a tile on the grid is updated, viewers of the tile will be notified of the update and can react to the changes. The idea is to allow you to build more complex, reactive systems on top of the grid.
- **Zero-allocation** (or close to it) traversal of the grid. The grid is pre-allocated entirely and this library provides a few ways of traversing it.
- **Path-finding**. The library provides a A\* pathfinding algorithm in order to compute a path between two points, as well as a BFS-based position scanning which searches the map around a point.

_Disclaimer_: the API or the library is not final and likely to change. Also, since this is just a side project of mine, don't expect this to be updated very often but please contribute!

# Grid & Tiles

The main entry in this library is `Grid` which represents, as the name implies a 2 dimentional grid which is the container of `Tile` structs. The `Tile` is essentially a cursor to a particular x,y coordinate and contains the following

- Value `uint32` of the tile, that can be used for calculating navigation or quickly retrieve sprite index.
- State set of `T comparable` that can be used to add additional information such as objects present on the tile. These things cannot be used for pathfinding, but can be used as an index.

Granted, uint32 value a bit small. The reason for this is the data layout, which is organised in thread-safe pages of 3x3 tiles, with the total size of 64 bytes which should neatly fit onto a cache line of a CPU.

In order to create a new `Grid[T]`, you first need to call `NewGridOf[T]()` method which pre-allocates the required space and initializes the tile grid itself. For example, you can create a 1000x1000 grid as shown below. The type argument `T` sets the type of the state objects. In the example below we want to create a new grid with a set of strings.

```go
grid := tile.NewGridOf[string](1000, 1000)
```

The `Each()` method of the grid allows you to iterate through all of the tiles in the grid. It takes an iterator function which is then invoked on every tile.

```go
grid.Each(func(p Point, t tile.Tile[string]) {
    // ...
})
```

The `Within()` method of the grid allows you to iterate through a set of tiles within a bounding box, specified by the top-left and bottom-right points. It also takes an iterator function which is then invoked on every tile matching the filter.

```go
grid.Within(At(1, 1), At(5, 5), func(p Point, t tile.Tile[string]) {
    // ...
})
```

The `At()` method of the grid allows you to retrieve a tile at a specific `x,y` coordinate. It simply returns the tile and whether it was found in the grid or not.

```go
if tile, ok := grid.At(50, 100); ok {
    // ...
}
```

The `WriteAt()` method of the grid allows you to update a tile at a specific `x,y` coordinate. Since the `Grid` itself is thread-safe, this is the way to (a) make sure the tile update/read is not racing and (b) notify observers of a tile update (more about this below).

```go
grid.WriteAt(50, 100, tile.Value(0xFF))
```

The `Neighbors()` method of the grid allows you to get the direct neighbors at a particular `x,y` coordinate and it takes an iterator funcion which is called for each neighbor. In this implementation, we are only taking direct neighbors (top, left, bottom, right). You rarely will need to use this method, unless you are rolling out your own pathfinding algorithm.

```go
grid.Neighbors(50, 100, func(point tile.Point, t tile.Tile[string]) {
    // ...
})
```

The `MergeAt()` method of the grid allows you to atomically update a value given a current value of the tile. For example, if we want to increment the value of a tile we can call this method with a function that increments the value. Under the hood, the increment will be done using an atomic compare-and-swap operation.

```go
grid.MergeAt(50, 100, func(v Value) Value {
    v += 1
    return v
})
```

The `MaskAt()` method of the grid allows you to atomically update only some of the bits at a particular `x,y` coordinate. This operation is as well thread-safe, and is actually useful when you might have multiple goroutines updating a set of tiles, but various goroutines are responsible for the various parts of the tile data. You might have a system that updates only a first couple of tile flags and another system updates some other bits. By using this method, two goroutines can update the different bits of the same tile concurrently, without erasing each other's results, which would happen if you just call `WriteAt()`.

```go
// assume byte[0] of the tile is 0b01010001
grid.MaskAt(50, 100,
    0b00101110, // Only last 2 bits matter
    0b00000011 // Mask specifies that we want to update last 2 bits
)

// If the original is currently: 0b01010001
// ...the result result will be: 0b01010010
```

# Pathfinding

As mentioned in the introduction, this library provides a few grid search / pathfinding functions as well. They are implemented as methods on the same `Grid` structure as the rest of the functionnality. The main difference is that they may require some allocations (I'll try to minimize it further in the future), and require a cost function `func(Tile) uint16` which returns a "cost" of traversing a specific tile. For example if the tile is a "swamp" in your game, it may cost higher than moving on a "plain" tile. If the cost function returns `0`, the tile is then considered to be an impassable obstacle, which is a good choice for walls and such.

The `Path()` method is used for finding a way between 2 points, you provide it the from/to point as well as costing function and it returns the path, calculated cost and whether a path was found or not. Note of caution however, avoid running it between 2 points if no path exists, since it might need to scan the entire map to figure that out with the current implementation.

```go
from := At(1, 1)
goal := At(7, 7)
path, distance, found := m.Path(from, goal, func(v tile.Value) uint16{
    if isImpassable(v) {
        return 0
    }
    return 1
})
```

The `Around()` method provides you with the ability to do a breadth-first search around a point, by providing a limit distance for the search as well as a cost function and an iterator. This is a handy way of finding things that are around the player in your game.

```go
point  := At(50, 50)
radius := 5
m.Around(point, radius, func(v tile.Value) uint16{
    if isImpassable(v) {
        return 0
    }
    return 1
}, func(p tile.Point, t tile.Tile[string]) {
    // ... tile found
})
```

# Observers

Given that the `Grid` is mutable and you can make changes to it from various goroutines, I have implemented a way to "observe" tile changes through a `NewView()` method which creates an `Observer` and can be used to observe changes within a bounding box. For example, you might want your player to have a view port and be notified if something changes on the map so you can do something about it.

In order to use these observers, you need to first call the `NewView()` function and start polling from the `Inbox` channel which will contain the tile update notifications as they happen. This channel has a small buffer, but if not read it will block the update, so make sure you always poll everything from it. Note that `NewView[S, T]` takes two type parameters, the first one is the type of the state object and the second one is the type of the tile value. The state object is used to store additional information about the view itself, such as the name of the view or a pointer to a socket that is used to send updates to the client.

In the example below we create a new 20x20 view on the grid and iterate through all of the tiles in the view.

```go
view := tile.NewView[string, string](grid, "My View #1")
view.Resize(tile.NewRect(0, 0, 20, 20), func(p tile.Point, t tile.Tile){
    // Optional, all of the tiles that are in the view now
})

// Poll the inbox (in reality this would need to be with a select, and a goroutine)
for {
    update := <-view.Inbox
    // Do something with update.Point, update.Tile
}
```

The `MoveBy()` method allows you to move the view in a specific direction. It takes in a `x,y` vector but it can contain negative values. In the example below, we move the view upwards by 5 tiles. In addition, we can also provide an iterator and do something with all of the tiles that have entered the view (e.g. show them to the player).

```go
view.MoveBy(0, 5, func(p tile.Point, tile tile.Tile){
    // Every tile which entered our view
})
```

Similarly, `MoveAt()` method allows you to move the view at a specific location provided by the coordinates. The size of the view stays the same and the iterator will be called for all of the new tiles that have entered the view port.

```go
view.MoveAt(At(10, 10), func(p tile.Point, t tile.Tile){
    // Every tile which entered our view
})
```

The `Resize()` method allows you to resize and update the view port. As usual, the iterator will be called for all of the new tiles that have entered the view port.

```go
viewRect := tile.NewRect(10, 10, 30, 30)
view.Resize(viewRect, func(p tile.Point, t tile.Tile){
    // Every tile which entered our view
})
```

The `Close()` method should be called when you are done with the view, since it unsubscribes all of the notifications. Be careful, if you do not close the view when you are done with it, it will lead to memory leaks since it will continue to observe the grid and receive notifications.

```go
// Unsubscribe from notifications and close the view
view.Close()
```

# Save & Load

The library also provides a way to save the `Grid` to an `io.Writer` and load it from an `io.Reader` by using `WriteTo()` method and `ReadFrom()` function. Keep in mind that the save/load mechanism does not do any compression, but in practice you should [use to a compressor](https://github.com/klauspost/compress) if you want your maps to not take too much of the disk space - snappy is a good option for this since it's fast and compresses relatively well.

The `WriteTo()` method of the grid only requires a specific `io.Writer` to be passed and returns a number of bytes that have been written down to it as well if any specific error has occured. Below is an example of how to save the grid into a compressed buffer.

```go
// Prepare the output buffer and compressor
output := new(bytes.Buffer)
writer, err := flate.NewWriter(output, flate.BestSpeed)
if err != nil {
    // ...
}

defer writer.Close()            // Make sure we flush the compressor
_, err := grid.WriteTo(writer)  // Write the grid
if err != nil {
    // ...
}
```

The `ReadFrom()` function allows you to read the `Grid` from a particular reader. To complement the example above, the one below shows how to read a compressed grid using this function.

```go
// Prepare a compressed reader over the buffer
reader := flate.NewReader(output)

// Read the Grid
grid, err := ReadFrom(reader)
if err != nil{
    // ...
}
```

# Benchmarks

This library contains quite a bit of various micro-benchmarks to make sure that everything stays pretty fast. Feel free to clone and play around with them yourself. Below are the benchmarks which we have, most of them are running on relatively large grids.

```
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkGrid/each-8                 868    1358434 ns/op           0 B/op   0 allocs/op
BenchmarkGrid/neighbors-8       66551679      17.87 ns/op           0 B/op   0 allocs/op
BenchmarkGrid/within-8             27207      44753 ns/op           0 B/op   0 allocs/op
BenchmarkGrid/at-8             399067512      2.994 ns/op           0 B/op   0 allocs/op
BenchmarkGrid/write-8          130207965      9.294 ns/op           0 B/op   0 allocs/op
BenchmarkGrid/merge-8          124156794      9.663 ns/op           0 B/op   0 allocs/op
BenchmarkGrid/mask-8           100000000      10.67 ns/op           0 B/op   0 allocs/op
BenchmarkState/range-8          12106854      98.91 ns/op           0 B/op   0 allocs/op
BenchmarkState/add-8            48827727      25.43 ns/op           0 B/op   0 allocs/op
BenchmarkState/del-8            52110474      21.59 ns/op           0 B/op   0 allocs/op
BenchmarkPath/9x9-8               264586        4656 ns/op      16460 B/op   3 allocs/op
BenchmarkPath/300x300-8              601     1937662 ns/op    7801502 B/op   4 allocs/op
BenchmarkPath/381x381-8              363     3304134 ns/op   62394356 B/op   5 allocs/op
BenchmarkPath/384x384-8              171     7165777 ns/op   62394400 B/op   5 allocs/op
BenchmarkPath/3069x3069-8             31    36479106 ns/op  124836075 B/op   4 allocs/op
BenchmarkPath/3072x3072-8             30    34889740 ns/op  124837686 B/op   4 allocs/op
BenchmarkPath/6144x6144-8            142     7594013 ns/op   62395376 B/op   3 allocs/op
BenchmarkAround/3r-8              506857        2384 ns/op        385 B/op   1 allocs/op
BenchmarkAround/5r-8              214280        5539 ns/op        922 B/op   2 allocs/op
BenchmarkAround/10r-8              85723       14017 ns/op       3481 B/op   2 allocs/op
BenchmarkPoint/within-8       1000000000      0.2190 ns/op          0 B/op   0 allocs/op
BenchmarkPoint/within-rect-8  1000000000      0.2195 ns/op          0 B/op   0 allocs/op
BenchmarkStore/save-8              14577       82510 ns/op          8 B/op   1 allocs/op
BenchmarkStore/read-8               3199      364771 ns/op     647419 B/op   7 allocs/op
BenchmarkView/write-8            6285351       188.2 ns/op         48 B/op   1 allocs/op
BenchmarkView/move-8               10000      116953 ns/op          0 B/op   0 allocs/op
```

# Contributing

We are open to contributions, feel free to submit a pull request and we'll review it as quickly as we can. This library is maintained by [Roman Atachiants](https://www.linkedin.com/in/atachiants/)

## License

Tile is licensed under the [MIT License](LICENSE.md).
