# Concurrent Pool

This is an implementation of a fixed size concurrent pool.

## Usage

The items in the pool must implement _io.Closer_ interface.

```go
    p, _ := pool.New(factory, 10, 5)   // create a pool
    p.Fill()                           // fill the pool
    item, _ := p.Get(context.TODO())   // get an item
    ...
    p.Put(item)                        // return it back to the pool
```