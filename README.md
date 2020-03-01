# Concurrent Pool

This is an implementation of a concurrent pool. Different from sync.Pool, the pool has a fix size.

## Usage

```go
    p, _ := pool.New(factory, 10, 5)   // create a pool
    p.Fill()                           // fill the pool
    item, _ := p.Get(context.TODO())   // get an item
    ...
    p.Put(item)                        // return it back to the pool
```