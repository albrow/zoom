Zoom
====

Version: X.X.X

A blazing-fast, lightweight ORM for Go built on Redis.

Full documentation is available on
[godoc.org](http://godoc.org/github.com/stephenalexbrowne/zoom).

**WARNING:** this isn't done yet and may change significantly before the official release. I do not
advise using Zoom for production or mission-critical applications. Feedback and pull requests are welcome :)

Since the development branch is quickly changing, this README might not necessarily reflect the most
up-to-date version of the API.


Philosophy
----------

If you want to build a high-performing, low latency application, but still want some of the ease
of an ORM, Zoom is for you!

Zoom allows you to:

- Persistently save structs of any type
- Retrieve structs from the database
- Preserve relationships between structs
- Preform *limited* SQL-like queries

Zoom consciously makes the trade off of using more memory in order to increase performance. However,
you can configure it to use less memory if you want. Zoom does not do sharding (but might in the future),
so be aware that memory could be a hard constraint for larger applications.

Zoom is a high-level library and abstracts away more complicated aspects of the Redis API. For example,
it manages its own connection pool, performs transactions when possible, and automatically converts
structs to and from a format suitable for the database. If needed, you can still execute redis commands
directly.

If you want to use advanced or complicated SQL queries, Zoom is not for you.

Installation
------------

First, you must install Redis on your system. [See installation instructions](http://redis.io/download).
By default, Zoom will use a tcp/http connection on localhost:6379 (same as the Redis default).

To install Zoom itself:

    go get github.com/stephenalexbrowne/zoom
    
This will pull the current master branch, which is (most likely) working but is quickly changing.

Getting Started
---------------

### Set Up

First, add github.com/stephenalexbrowne/zoom to your import statement:

``` go
import (
    ...
    github.com/stephenalexbrowne/zoom
)
```

Then, call zoom.Init somewhere in your app initialization code, e.g. in the main function. You must
also call zoom.Close when your application exits, so it's a good idea to use defer.

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

The Init function takes a *zoom.Configuration struct as an argument. Here's a list of options and their
defaults:

``` go
type Configuration struct {
	Address       string // Address to connect to. Default: "localhost:6379"
	Network       string // Network to use. Default: "tcp"
	Database      int    // Database id to use (using SELECT). Default: 0
}
```

If possible, it is ***strongly recommended*** that you use a unix socket connection instead of tcp.
Redis is roughly [50% faster](http://redis.io/topics/benchmarks) this way. To connect with a unix
socket, you must first configure redis to accept socket connections (typically on /tmp/redis.sock).
If you are unsure how to do this, refer to the [official redis docs](http://redis.io/topics/config)
for help. You might also find the [redis quickstart guide](http://redis.io/topics/quickstart) helpful,
especially the bottom sections.

To use unix sockets with Zoom, simply pass in "unix" as the Network and "/tmp/unix.sock" as the Address:

``` go
config := &zoom.Configuration {
	Address: "/tmp/redis.sock",
	Network: "unix",
}
if err := zoom.Init(config); err != nil {
	// handle err
}
```

### Creating Models

In order to save a struct using Zoom, you need to embed an anonymous DefaultData field. DefaultData
gives you an Id and getters and setters for that id, which are required in order to save the model
to the database. Here's an example of a Person model:

``` go
type Person struct {
    Name String
    zoom.DefaultData
}
```

You must also call zoom.Register so that Zoom knows what name to assign to a given type. You only
need to do this once per type. For example, somewhere in your initialization sequence (e.g. in the main
function) put the following:

``` go
// register the *Person type as "person"
if err := zoom.Register(&Person{}, "person"); err != nil {
    // handle error
}
```

### Saving Models

To persistently save a Person model to the databse, simply call zoom.Save.

``` go
p := NewPerson("Alice")
if err := zoom.Save(p); err != nil {
    // handle error
}
```

### Deleting Models

To delete a model you can just use the Delete function:

``` go
if err := zoom.Delete(person); err != nil {
	// handle err
}
```

Or if you know the model's id, use the DeleteById function:

``` go
if err := zoom.DeleteById("some_person_id"); err != nil {
	// handle err
}
```

Running Queries
---------------

### The Query Object

Zoom provides a useful abstraction for querying the database. Queries read nicely and are easy to write.
The Query interface consists of only one method:

``` go
type Query interface {
	Run() (interface{}, error)
}
```

There are two different types of queries satisfying the query interface: ModelQuery and MultiModelQuery.
ModelQuery is used for finding a single record, while MultiModelQuery is used for finding one or more
records. There are two different constructors for each type of query, one (the Find* constructor) involves
depending on a type-unsafe return value, and (the Scan* constructor) the other requires passing in a pointer
to a scannable Model or array of models.

### Finding a Single Model

Zoom will automatically assign a random, unique id to each saved model. To retrieve a model by id,
create a ModelQuery object using the FindById or ScanById constructor. Then call Run() on the query object.

Here's an example of using FindById. You must use the same string name you used in zoom.Register.
The return type of Run is interface{} so you may need to type assert.

``` go
result, err := zoom.FindById("person", "a_valid_person_id").Run()
if err != nil {
    // handle error
}

// type assert
person, ok := result.(*Person)
if !ok {
    // handle !ok
}
```

The ScanById constructor expects a pointer to a struct of registered type as an argument. It is
sometimes easier to use because it doesn't require type assertion.

``` go
p := &Person{}
if err := zoom.ScanById("a_valid_person_id", p).Run(); err != nil {
    // handle error
}
```

### Finding One or More Models of a Given Type 

To retrieve a list of all persons, we need a FindAll query. You can use either the FindAll constructor
or the ScanAll constructor.

Here's an example of using FindAll. You must use the same string name you used in zoom.Register.
The return type of Run is interface{}, but the underlying type is a slice of Model, i.e. a slice
of pointers to structs. You may need a type assertion.

``` go
results, err := zoom.FindAll("person").Run()
if err != nil {
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

You can use the ScanAll constructor if you want to avoid a type assertion. ScanAll expects a pointer to
a slice or array of Model.

``` go
persons := make([]*Person, 0)
if _, err := zoom.ScanAll(persons).Run(); err != nil {
	// handle err
}
```

### Using Query Modifiers

You can chain a query object together with one or more different modifiers and then call Run
when you are ready to run the query. 

The modifiers for a ModelQuery are: Include and Exclude.

The modifiers for a MultiModelQuery are: Include, Exclude, SortBy, Order, Limit, and Offset. 

Here's an example of a more complicated query using several modifiers:

``` go
q := zoom.FindAll("person").SortBy("Name").Order("DESC").Limit(10)
result, err := q.Run()
```

Full documentation on the different modifiers is available on
[godoc.org](http://godoc.org/github.com/stephenalexbrowne/zoom).


Relationships
-------------

Relationships in Zoom are simple. There are no special return types or functions for using relationships.
What you put in is what you get out.

### One-to-One Relationships

For these examples we're going to introduce two new struct types:

``` go
// The PetOwner struct
type PetOwner struct {
	Name  string
	Pet   *Pet    // *Pet should also be a registered type
	zoom.DefaultData
}

// The Pet struct
type Pet struct {
	Name   string
	zoom.DefaultData
}

```

Assuming you've registered both the *PetOwner and *Pet types, Zoom will automatically set up a relationship
when you save a PetOwner with a valid Pet. (The Pet must have an id)

``` go
// create a new PetOwner and a Pet
owner := &PetOwner{Name: "Bob"}
pet := &Pet{Name: "Spot"}

// save the pet
if err := zoom.Save(pet); err != nil {
	// handle err
}

// set the owner's pet
owner.Pet = pet

// save the owner
if err := zoom.Save(owner); err != nil {
	// handle err
}
```

Now if you retrieve the pet owner by it's id, the pet attribute will persist as well.

For now, Zoom does not support reflexivity of one-to-one relationships. So if you want pet ownership to be
bidirectional (i.e. if you want an owner to know about its pet **and** a pet to know about its owner),
you would have to manually set up both relationships.

``` go
ownerCopy := &PetOwner{}
q := zoom.ScanById("the_id_of_above_pet_owner", ownerCopy)
if _, err := q.Run(); err != nil {
	// handle err
}

// the Pet attribute is still set
fmt.Println(ownerCopy.Pet.Name)

// Output:
//	Spot
```

### One-to-Many Relationships

One-to-many relationships work similarly. This time we're going to use two new struct types in the examples.

``` go
// The Parent struct
type Parent struct {
	Name     string
	Children []*Child  // *Child should also be a registered type
	zoom.DefaultData
}

// The Child struct
type Child struct {
	Name   string
	zoom.DefaultData
}
```

Assuming you register both the *Parent and *Child types, Zoom will automatically set up a relationship
when you save a parent with some children (as long as each child has an id). Here's an example:

``` go
// create a parent and two children
parent := &Parent{Name: "Christine"}
child1 := &Child{Name: "Derick"}
child2 := &Child{Name: "Elise"}

// save the children
if err := zoom.Save(child1, child2); err != nil {
	// handle err
}

// assign the children to the parent
parent.Children = append(parent.Children, child1, child2)

// save the parent
if err := zoom.Save(parent); err != nil {
	// handle err
}
```

Again, Zoom does not support reflexivity. So if you wanted a child to know about its parent, you would have
to set up and manage the relationship manually. This might change in the future.

Now when you retrieve a parent by id, it's children field will automatically be populated. So getting
the children again is straight forward.

``` go
parentCopy := &Parent{}
q := zoom.ScanById("the_id_of_above_parent", parentCopy)
if _, err := q.Run(); err != nil {
	// handle error
}

// now you can access the children normally
for _, child := range parentCopy.Children {
	fmt.Println(child.Name)
}

// Output:
//	Derick
//	Elise

```

For a Parent with a lot of children, it may take a long time to get each Child from the database. If
this is the case, it's a good idea to use the Exclude modifier when you don't intend to use the children.

``` go
parentCopyNoChildren := &Parent{}
q := zoom.ScanById("the_id_of_above_parent", parentCopyNoChildren).Exclude("Children")
if _, err := q.Run(); err != nil {
	// handle error
}

// Since it was excluded, Children is empty.
fmt.Println(parentCopyNoChildren.Children)

// Output:
//	[]
```

### Many-to-Many Relationships

There is nothing special about many-to-many relationships. They are simply made up of multiple one-to-many
relationships.


Testing & Benchmarking
----------------------

**IMPORTANT**: Before running any tests or benchmarks, make sure you have a redis-server instance running.
The tests and benchmarks will attempt to use a socket connection on /tmp/redis.sock. If that doesn't work,
they will fallback to a tcp connection on localhost:6379.

All the tests and benchmarks will use database #9. If database #9 is non-empty, they will
will throw and error and not run. (so as to not corrupt your data). Database #9 is flushed at the end of
every test/benchmark.

### Running the Tests:

To run the tests, make sure you're in the root directory for Zoom and run:

    go test ./...
    
If everything passes, you should see something like:

    ?       github.com/stephenalexbrowne/zoom   [no test files]
    ok      github.com/stephenalexbrowne/zoom/redis 1.338s
    ok      github.com/stephenalexbrowne/zoom/test  0.367s
    ?       github.com/stephenalexbrowne/zoom/test_support  [no test files]
    ok      github.com/stephenalexbrowne/zoom/util  0.101s
    
If any of the tests fail, please [open an issue](https://github.com/stephenalexbrowne/zoom/issues/new) and
describe what happened.

### Running the Benchmarks:

To run the benchmarks, again make sure you're in the root directory and run:

    go test ./... -bench .
    
You should see some runtimes for various operations. If you see an error or if the build fails, please
[open an issue](https://github.com/stephenalexbrowne/zoom/issues/new).

Here are the results from my laptop (2.3GHz intel i7, 8GB ram):

```
BenchmarkConnection      20000000          99.2 ns/op
BenchmarkPing               50000         40761 ns/op
BenchmarkSet                50000         48653 ns/op
BenchmarkGet                50000         42537 ns/op
BenchmarkSave               50000         70305 ns/op
BenchmarkFindById           50000         55120 ns/op
BenchmarkScanById           50000         54960 ns/op
BenchmarkFindByIdExclude    50000         57914 ns/op
BenchmarkRepeatDeleteById   50000         63256 ns/op
BenchmarkRandomDeleteById   50000         64386 ns/op
BenchmarkFindAll10          10000        208239 ns/op
BenchmarkFindAll100          2000        987973 ns/op
BenchmarkFindAll1000          200       7915705 ns/op
BenchmarkFindAll10000          20      83554133 ns/op
BenchmarkScanAll10          10000        204737 ns/op
BenchmarkScanAll100          2000        935378 ns/op
BenchmarkScanAll1000          200       7834456 ns/op
BenchmarkScanAll10000          20      84332092 ns/op
BenchmarkSortNumeric10      10000        219864 ns/op
BenchmarkSortNumeric100      2000       1022086 ns/op
BenchmarkSortNumeric1000      200       8935983 ns/op
BenchmarkSortNumeric10000      20      93927469 ns/op
BenchmarkSortNumeric10000Limit1      100      11256255 ns/op
BenchmarkSortNumeric10000Limit10     100      11289648 ns/op
BenchmarkSortNumeric10000Limit100    100      11982038 ns/op
BenchmarkSortNumeric10000Limit1000    50      20232813 ns/op
BenchmarkSortAlpha10       10000        225211 ns/op
BenchmarkSortAlpha100       2000       1074775 ns/op
BenchmarkSortAlpha1000       200       9372886 ns/op
BenchmarkSortAlpha10000       20     100336418 ns/op
```

You should run your own benchmarks that are closer to your use case to get a real sense of how Zoom
will perform for you. The speeds above are already pretty fast, but improving them is one of the top
priorities for this project.
    
Example Usage
-------------

The [zoom_example repository](https://github.com/stephenalexbrowne/zoom_example) is an up-to-date
example of how to use Zoom in a json/rest application.


TODO
----

Ordered generally by priority, here's what I'm working on:

- Add a --host flag to benchmarks and tests
- Improve sort/limit/offset performance by using custom indeces
- Add Filter and Count modifiers to MultiModelQuery
- Support AND and OR operators on Filters
- Support combining queries into a single transaction
- Use scripting to reduce round-trip latencies in queries
- Implement high-level watching for record changes
- Support callbacks (BeforeSave, AfterSave, BeforeDelete, AfterDelete, etc.)
- Add option to make relationships reflexive (inverseOf struct tag?)
- Add a dependent:delete struct tag
- Support automatic sharding


LICENSE
-------

Zoom is licensed under the MIT License. See the LICENSE file for more information.

The redis driver for Zoom is based on https://github.com/garyburd/redigo and is licensed under
the Apache License, Version 2.0. See the NOTICE file for more information. Thanks Gary!
