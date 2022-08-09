This repository contains an implementation of the [returnFrom](https://github.com/golang/go/issues/54361) proposal implemented by syntax rewriting.

### Usage

To try it out on a single file:

```sh
go run github.com/ConradIrwin/return-from example.go > example-rewritten.go
go run example-rewritten.go
```

If you want to try many examples, you can also clone the repo and use `rewrite-run.sh` which auto-runs the rewritten code:

```sh
git clone https://github.com/ConradIrwin/return-from
cd return-from
./rewrite-run.sh example.go
```

You can see a few examples in the ./examples folder.

### Limitations

This does not yet work well with generics, in particular:

- You cannot returnFrom(x) if you are inside a generic function named x.
- If you want to returnFrom(x) where x is a call to a generic function, you must specify the type parameter (this should not be the case).
- Not very well tested yet, please report bugs!

### License

As these are exampels you may use them for whatever purpose you like with no restrictions or warranties of any kind.

This code is also available under the MIT license or the CC0 license.
