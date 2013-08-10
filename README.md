Zoom
====

A blazing-fast, lightweight ORM-ish library for go and redis.

Why "ORM-ish" instead of just "ORM"? Go doesn't really have objects in the traditional sense.
Perhaps it would be more accurate to call it SRM (Struct Relational Mapping), but no one would really
know what that acronym stands for.

**WARNING:** this isn't done yet and may change significantly before the official release. I do not
advise using Zoom for production or mission-critical applications. However, you are free to inspect
the code and play around with it.


Philosophy
----------

Zoom allows you to:

- Persistently save structs of any type
- Retrieve structs from the database
- Preserve relationships between structs (The "R" in ORM)

Zoom, like the Go language, is intended to be minimal. It is a light-weight ORM with a clear set of goals.
It does what it's supposed to and it also does it ***very fast***.


Installation
------------

First, you must install redis on your system. [See installation instructions](http://redis.io/download).

By default, Zoom uses a unix socket connection to connect to redis. To do this, you need to enable
socket connections in your redis config file. If you prefer to use tcp/http instead, see the Setup
instructions below.  

To install Zoom itself:

    go get github.com/stephenalexbrowne/zoom
    
This will pull the current master branch, which is (most likely) stable but is quickly changing.

Getting Started
-----

### Setup

First, add github.com/stephenalexbrowne/zoom to your import statement:

``` go
import (
    ...
    github.com/stephenalexbrowne/zoom
)
```

Then, call Zoom.Init() somewhere in your app initialization code, e.g. in the main() method. You must
also call Zoom.Close() when your application exits, so it's a good idea to use defer.

``` go
func main() {
    // ...
    if err := zoom.Init(nil); err != nil {
        // handle err
    }
    defer zoom.Close()
    // ...
}
```

The Init() method takes a *zoom.Configuration struct as an argument. Here's a list of options and their
defaults:

``` go
type Configuration struct {
    Address  string // Address to connect to.   Default: "/tmp/redis.sock"
	Network  string // Network to use.          Default: "unix"
	Database int    // Database id to use.      Default: 0
}
```

So, if you wanted to connect with tcp/http on the default port (6380), you would do this:

``` go
config := &zoom.Configuration {
    Address: "localhost:6380",
    Network: "tcp",
}
if err := zoom.Init(config); err != nil {
    // handle err
}
```

### Creating your models

In order to save a struct using Zoom, you need to add an embedded struct attribute. Throughout the rest of
this guide, we'll be using a Person struct as an example:

``` go
type Person struct {
    Name String
    *zoom.Model
}
```

The *zoom.Model embedded attribute automatically gives you an Id field. You do not need to add an Id field
to the struct.

**Important:** In the constructor, you must initialize the embedded *zoom.Model attribute. Something like this:

``` go
func NewPerson(name string) *Person {
    return &Person{
        Name:  name,
        Model: new(zoom.Model), // don't forget this!
    }
}
```

You must also call zoom.Register() so that Zoom knows what name to assign to the *Person type. You only
need to do this once per type. For example, somewhere in your initialization sequence (e.g. in the main()
method) put the following:

``` go
// register the *Person type as "person"
if err := zoom.Register(&Person{}, "person"); err != nil {
    // handle error
}
```

### Saving to redis

To persistently save a Person model to the databse, you would simply call zoom.Save()

``` go
p := NewPerson("Alice")
if err := zoom.Save(p); err != nil {
    // handle error
}
```

### Retreiving from redis

Zoom will automatically assign a random, unique id to each saved model. To retrieve a model by id,
you must use the same string name you used in zoom.Register. The return type
is interface{}, so you may need to cast to *Person using a type assertion.

``` go
if result, err := zoom.FindById("person", "your-person-id"); err != nil {
    // handle error
}

// type assert
if person, ok := result.(*Person); !ok {
    // handle !ok
}
```
    
To retrieve a list of all persons use zoom.FindAll(). Like FindById() the return type is []interface{}.
If you want an array or slice of *Person, you need to type assert each element individually.

``` go
if results, err := zoom.FindAll("person"); err != nil {
    // handle error
}

// type assert each element. something like:
persons := make([]*Person, len(results))
for i, result := range results {
    if person, ok := result.(*Person); !ok {
        // handle !ok
    }
    persons[i] = person
}
```

Testing & Benchmarking
----------------------

**Important:** Before running any tests or benchmarks, make sure that you have redis running and accepting
connections on a unix socket at /tmp/unix.sock. To test if your redis server is properly set up, you can run:

    redis-cli -s /tmp/redis.sock -c ping
    
If you receive PONG in response, then you are good to go. If anything else happens, redis is not setup
properly. Check out the [official redis docs](http://redis.io/topics/config) for help. You might also find
the [redis quickstart guide](http://redis.io/topics/quickstart) helpful, especially the bottom sections.

### Running the Tests:

To run the tests, make sure you're in the project root directory and run:

    go test ./...
    
If everything passes, you should see something like:

    ?   	github.com/stephenalexbrowne/zoom	[no test files]
    ok  	github.com/stephenalexbrowne/zoom/benchmark	0.147s
    ok  	github.com/stephenalexbrowne/zoom/redis	0.272s
    ok  	github.com/stephenalexbrowne/zoom/test	0.155s
    ok  	github.com/stephenalexbrowne/zoom/test_relate	0.277s
    
If any of the tests fail, please [open an issue](https://github.com/stephenalexbrowne/zoom/issues/new) and
describe what happened.

### Running the Benchmarks:

To run the benchmarks, again make sure you're in the project root directory and run:

    go test ./... --bench="."
    
You should see some runtimes for various operations. If you see an error or if the build fails, please
[open an issue](https://github.com/stephenalexbrowne/zoom/issues/new).

Here are the results from my laptop (2.3GHz intel i7, 8G ram):

```
BenchmarkSave	   20000	     99562 ns/op
BenchmarkFindById	   50000	     73934 ns/op
BenchmarkDeleteById	   50000	     70942 ns/op
```

To put the results another way: 

- Writes take about 100 microseconds (0.01 ms)
- You can perform about 10k writes/second
- Reads take about 75 microseconds (0.075 ms)
- You can perform about 13.5k writes/second

That's already pretty fast! And improving these speeds is one of the top priorities for this project.

    
Example Usage
-------------

The [zoom_example repository](https://github.com/stephenalexbrowne/zoom_example) is an up-to-date example
of how to use Zoom in a json/rest application. Use it as a reference if anything above is not clear. Formal
documentation is on my todo list.


LICENSE
-------

Zoom is licensed under the MIT License. See the LICENSE file for more information.

The redis driver for Zoom is based on https://github.com/garyburd/redigo and is licensed under
the Apache License, Version 2.0. See redis/NOTICE for more information. Thanks Gary!
