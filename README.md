Zoom
====

[![Version](https://img.shields.io/badge/version-0.18.0-5272B4.svg)](https://github.com/albrow/zoom/releases)
[![Circle CI](https://img.shields.io/circleci/project/albrow/zoom/master.svg)](https://circleci.com/gh/albrow/zoom/tree/master)
[![GoDoc](https://godoc.org/github.com/albrow/zoom?status.svg)](https://godoc.org/github.com/albrow/zoom)

A blazing-fast datastore and querying engine for Go built on Redis.

Requires Redis version >= 2.8.9 and Go version >= 1.2. The latest version of
both is recommended.

Full documentation is available on
[godoc.org](http://godoc.org/github.com/albrow/zoom).


Table of Contents
-----------------

<!-- toc -->

- [Development Status](#development-status)
- [When is Zoom a Good Fit?](#when-is-zoom-a-good-fit)
- [Installation](#installation)
- [Initialization](#initialization)
- [Models](#models)
  * [What is a Model?](#what-is-a-model)
  * [Customizing Field Names](#customizing-field-names)
  * [Creating Collections](#creating-collections)
  * [Saving Models](#saving-models)
  * [Updating Models](#updating-models)
  * [Finding a Single Model](#finding-a-single-model)
  * [Finding Only Certain Fields](#finding-only-certain-fields)
  * [Finding All Models](#finding-all-models)
  * [Deleting Models](#deleting-models)
  * [Counting the Number of Models](#counting-the-number-of-models)
- [Transactions](#transactions)
- [Queries](#queries)
  * [The Query Object](#the-query-object)
  * [Using Query Modifiers](#using-query-modifiers)
  * [A Note About String Indexes](#a-note-about-string-indexes)
- [More Information](#more-information)
  * [Persistence](#persistence)
  * [Atomicity](#atomicity)
  * [Concurrent Updates and Optimistic Locking](#concurrent-updates-and-optimistic-locking)
- [Testing & Benchmarking](#testing--benchmarking)
  * [Running the Tests](#running-the-tests)
  * [Running the Benchmarks](#running-the-benchmarks)
- [Contributing](#contributing)
- [Example Usage](#example-usage)
- [License](#license)

<!-- tocstop -->

Development Status
------------------

Zoom was first started in 2013. It is well-tested and going forward the API
will be relatively stable. However, it is not actively maintained and there
are some known performance issues with queries that use more than one filter.

At this time, Zoom can be considered safe for use in low-traffic production
applications. However, I would recommend that you look at more actively
maintained alternatives.

Zoom follows semantic versioning, but offers no guarantees of backwards
compatibility until version 1.0. You can also keep an eye on the
[Releases page](https://github.com/albrow/zoom/releases) to see a full changelog
for each release. In addition, starting with version 0.9.0,
[migration guides](https://github.com/albrow/zoom/wiki/Migration-Guide) will be
provided for any non-trivial breaking changes, making it easier to stay up to
date with the latest version.


When is Zoom a Good Fit?
------------------------

Zoom might be a good fit if:

1. **You are building a low-latency application.** Because Zoom is built on top of
	Redis and all data is stored in memory, it is typically much faster than datastores/ORMs
	based on traditional SQL databases. Latency will be the most noticeable difference, although
	throughput may also be improved.
2. **You want more out of Redis.** Zoom offers a number of features that you don't get
	by using a Redis driver directly. For example, Zoom supports a larger number of types
	out of the box (including custom types, slices, maps, complex types, and embedded structs),
	provides tools for making multi-command transactions easier, and of course, provides the
	ability to run queries.
3. **You want an easy-to-use datastore.** Zoom has a simple API and is arguably easier to
	use than some ORMs. For example, it doesn't require database migrations and instead builds
	up a schema based on your struct types. Zoom also does not typically require any knowledge
	of Redis in order to use effectively. Just connect it to a database and you're good to go!

Zoom might ***not*** be a good fit if:

1. **You are working with a lot of data.** Redis is an in-memory database, and Zoom does not
	yet support sharding or Redis Cluster. Memory could be a hard constraint for larger applications.
	Keep in mind that it is possible (if expensive) to run Redis on machines with up to 256GB of memory
	on cloud providers such as Amazon EC2.
2. **You need advanced queries.** Zoom currently only provides support for basic queries and is
	not as powerful or flexible as something like SQL. For example, Zoom currently lacks the
	equivalent of the `IN` or `OR` SQL keywords. See the
	[documentation](http://godoc.org/github.com/albrow/zoom/#Query) for a full list of the types
	of queries supported.


Installation
------------

Zoom is powered by Redis and needs to connect to a Redis database. You can install Redis on the same
machine that Zoom runs on, connect to a remote database, or even use a Redis-as-a-service provider such
as Redis To Go, RedisLabs, Google Cloud Redis, or Amazon Elasticache.

If you need to install Redis, see the [installation instructions](http://redis.io/download) on the official
Redis website.

To install Zoom itself, run `go get -u github.com/albrow/zoom` to pull down the
current master branch, or install with the dependency manager of your choice to
lock in a specific version.


Initialization
--------------

First, add github.com/albrow/zoom to your import statement:

``` go
import (
	 // ...
	 github.com/albrow/zoom
)
```

Then, you must create a new pool with 
[`NewPool`](http://godoc.org/github.com/albrow/zoom/#NewPool). A pool represents
a pool of connections to the database. Since you may need access to the pool in
different parts of your application, it is sometimes a good idea to declare a
top-level variable and then initialize it in the `main` or `init` function. You
must also call `pool.Close` when your application exits, so it's a good idea to
use defer.

``` go
var pool *zoom.Pool

func main() {
	pool = zoom.NewPool("localhost:6379")
	defer func() {
		if err := pool.Close(); err != nil {
			// handle error
		}
	}()
	// ...
}
```

The `NewPool` function accepts an address which will be used to connect to
Redis, and it will use all the
[default values](http://godoc.org/github.com/albrow/zoom/#DefaultPoolOptions)
for the other options. If you need to specify different options, you can use the
[`NewPoolWithOptions`](http://godoc.org/github.com/albrow/zoom/#NewPoolWithOptions)
function.

For convenience, the
[`PoolOptions`](http://godoc.org/github.com/albrow/zoom/#PoolOptions) type has
chainable methods for changing each option. Typically you would start with
[`DefaultOptions`](http://godoc.org/github.com/albrow/zoom/#DefaultOptions) and
call `WithX` to change value for option `X`.

For example, here's how you could initialize a Pool that connects to Redis using
a unix socket connection on `/tmp/unix.sock`:

``` go
options := zoom.DefaultPoolOptions.WithNetwork("unix").WithAddress("/tmp/unix.sock")
pool = zoom.NewPoolWithOptions(options)
```


Models
------

### What is a Model?

Models in Zoom are just structs which implement the `zoom.Model` interface:

``` go
type Model interface {
  ModelID() string
  SetModelID(string)
}
```

To clarify, all you have to do to implement the `Model` interface is add a getter and setter
for a unique id property.

If you want, you can embed `zoom.RandomID` to give your model all the
required methods. A struct with `zoom.RandomID` embedded will generate a pseudo-random id for itself
the first time the `ModelID` method is called iff it does not already have an id. The pseudo-randomly
generated id consists of the current UTC unix time with second precision, an incremented atomic
counter, a unique machine identifier, and an additional random string of characters. With ids generated
this way collisions are extremely unlikely.

Future versions of Zoom may provide additional id implementations out of the box, e.g. one that assigns
auto-incremented ids. You are also free to write your own id implementation as long as it satisfies the
interface.

A struct definition serves as a sort of schema for your model. Here's an example of a model for a person:

``` go
type Person struct {
	 Name string
	 Age  int
	 zoom.RandomID
}
```

Because of the way Zoom uses reflection, all the fields you want to save need to be exported.
Unexported fields (including unexported embedded structs with exported fields) will not
be saved. This is a departure from how the  encoding/json and  encoding/xml packages
behave. See [issue #25](https://github.com/albrow/zoom/issues/25) for discussion.

Almost any type of field is supported, including custom types, slices, maps, complex types,
and embedded structs. The only things that are not supported are recursive data structures and
functions.

### Customizing Field Names

You can change the name used to store the field in Redis with the `redis:"<name>"` struct tag. So
for example, if you wanted the fields to be stored as lowercase fields in Redis, you could use the
following struct definition:

``` go
type Person struct {
	 Name string    `redis:"name"`
	 Age  int       `redis:"age"`
	 zoom.RandomID
}
```

If you don't want a field to be saved in Redis at all, you can use the special struct tag `redis:"-"`.

### Creating Collections

You must create a `Collection` for each type of model you want to save. A
`Collection` is simply a set of all models of a specific type and has methods
for saving, finding, deleting, and querying those models. `NewCollection`
examines the type of a model and uses reflection to build up an internal schema.
You only need to call `NewCollection` once per type. Each pool keeps track of
its own collections, so if you wish to share a model type between two or more
pools, you will need to create a collection for each pool.

``` go
// Create a new collection for the Person type.
People, err := pool.NewCollection(&Person{})
if err != nil {
	 // handle error
}
```


The convention is to name the `Collection` the plural of the corresponding
model type (e.g. "People"), but it's just a variable so you can name it
whatever you want.

`NewCollection` will use all the
[default options](http://godoc.org/github.com/albrow/zoom/#DefaultCollectionOptions)
for the collection.

If you need to specify other options, use the
[`NewCollectionWithOptions`](http://godoc.org/github.com/albrow/zoom/#NewCollectionWithOptions)
function. The second argument to `NewCollectionWithOptions` is a
[`CollectionOptions`](http://godoc.org/github.com/albrow/zoom#CollectionOptions).
It works similarly to `PoolOptions`, so you can start with
[`DefaultCollectionOptions`](http://godoc.org/github.com/albrow/zoom/#DefaultCollectionOptions)
and use the chainable `WithX` methods to specify a new value for option `X`.

Here's an example of how to create a new `Collection` which is indexed, allowing
you to use Queries and methods like `FindAll` which rely on collection indexing:

``` go
options := zoom.DefaultCollectionOptions.WithIndex(true)
People, err = pool.NewCollectionWithOptions(&Person{}, options)
if err != nil {
	// handle error
}
```

There are a few important points to emphasize concerning collections:

1. The collection name cannot contain a colon.
2. Queries, as well as the `FindAll`, `DeleteAll`, and `Count` methods will not
	work if `Index` is `false`. This may change in future versions.

If you need to access a `Collection` in different parts of
your application, it is sometimes a good idea to declare a top-level variable
and then initialize it in the `init` function:

```go
var (
	People *zoom.Collection
)

func init() {
	var err error
	// Assuming pool and Person are already defined.
	People, err = pool.NewCollection(&Person{})
	if err != nil {
		// handle error
	}
}
```


### Saving Models

Continuing from the previous example, to persistently save a `Person` model to
the database, we use the `People.Save` method. Recall that in this example,
"People" is just the name we gave to the `Collection` which corresponds to the
model type `Person`.

``` go
p := &Person{Name: "Alice", Age: 27}
if err := People.Save(p); err != nil {
	 // handle error
}
```

When you call `Save`, Zoom converts all the fields of the model into a format
suitable for Redis and stores them as a Redis hash. There is a wiki page
describing
[how zoom works under the hood](https://github.com/albrow/zoom/wiki/Under-the-Hood) in more detail.

### Updating Models

Sometimes, it is preferable to only update certain fields of the model instead
of saving them all again. It is more efficient and in some scenarios can allow
safer simultaneous changes to the same model (as long as no two clients update
the same field at the same time). In such cases, you can use `UpdateFields`.

``` go
if err := People.UpdateFields([]string{"Name"}, person); err != nil {
	// handle error
}
```

`UpdateFields` uses "last write wins" semantics, so if another caller updates
the same field, your changes may be overwritten. That means it is not safe for
"read before write" updates. See the section on
[Concurrent Updates](#concurrent-updates-and-optimistic-locking) for more
information.

### Finding a Single Model

To retrieve a model by id, use the `Find` method:

``` go
p := &Person{}
if err := People.Find("a_valid_person_id", p); err != nil {
	 // handle error
}
```

The second argument to `Find` must be a pointer to a struct which satisfies `Model`, and must have a type corresponding to
the `Collection`. In this case, we passed in `Person` since that is the struct type that corresponds to our `People`
collection. `Find` will mutate `p` by setting all its fields. Using `Find` in this way allows the caller to maintain type
safety and avoid type casting. If Zoom couldn't find a model of type `Person` with the given id, it will return a
`ModelNotFoundError`.

### Finding Only Certain Fields

If you only want to find certain fields in the model instead of retrieving all
of them, you can use `FindFields`, which works similarly to `UpdateFields`.

``` go
p := &Person{}
if err := People.FindFields("a_valid_person_id", []string{"Name"}, p); err != nil {
	// handle error
}
fmt.Println(p.Name, p.Age)
// Output:
// Alice 0
```

Fields that are not included in the given field names will not be mutated. In
the above example, `p.Age` is `0` because `p` was just initialized and that's
the zero value for the `int` type.

### Finding All Models

To find all models of a given type, use the `FindAll` method:

``` go
people := []*Person{}
if err := People.FindAll(&people); err != nil {
	 // handle error
}
```

`FindAll` expects a pointer to a slice of some registered type that implements `Model`. It grows or shrinks the slice as needed,
filling in all the fields of the elements inside of the slice. So the result of the call is that `people` will be a slice of
all models in the `People` collection.

`FindAll` only works on indexed collections. To index a collection, you need to
include `Index: true` in the `CollectionOptions`.

### Deleting Models

To delete a model, use the `Delete` method:

``` go
// ok will be true iff a model with the given id existed and was deleted
if ok, err := People.Delete("a_valid_person_id"); err != nil {
	// handle err
}
```

`Delete` expects a valid id as an argument, and will attempt to delete the model with the given id. If there was no model
with the given type and id, the first return value will be false.

You can also delete all models in a collection with the `DeleteAll` method:

``` go
numDeleted, err := People.DeleteAll()
if err != nil {
  // handle error
}
```

`DeleteAll` will return the number of models that were successfully deleted.
`DeleteAll` only works on indexed collections. To index a collection, you need
to include `Index: true` in the `CollectionOptions`.

### Counting the Number of Models

You can get the number of models in a collection using the `Count` method:

``` go
count, err := People.Count()
if err != nil {
  // handle err
}
```

`Count` only works on indexed collections. To index a collection, you need
to include `Index: true` in the `CollectionOptions`.


Transactions
------------

Zoom exposes a transaction API which you can use to run multiple commands efficiently and atomically. Under the hood,
Zoom uses a single [Redis transaction](http://redis.io/topics/transactions) to perform all the commands in a single
round trip. Transactions feature delayed execution, so nothing touches the database until you call `Exec`. A transaction
also remembers its errors to make error handling easier on the caller. The first error that occurs (if any) will be
returned when you call `Exec`.

Here's an example of how to save two models and get the new number of models in
the `People` collection in a single transaction.

``` go
numPeople := 0
t := pool.NewTransaction()
t.Save(People, &Person{Name: "Foo"})
t.Save(People, &Person{Name: "Bar"})
// Count expects a pointer to an integer, which it will change the value of
// when the transaction is executed.
t.Count(People, &numPeople)
if err := t.Exec(); err != nil {
  // handle error
}
// numPeople will now equal the number of `Person` models in the database
fmt.Println(numPeople)
// Output:
// 2
```

You can execute custom Redis commands or run custom Lua scripts inside a
[`Transaction`](http://godoc.org/github.com/albrow/zoom/#Transaction) using the
[`Command`](http://godoc.org/github.com/albrow/zoom/#Transaction.Command) and
[`Script`](http://godoc.org/github.com/albrow/zoom/#Transaction.Script) methods.
Both methods expect a
[`ReplyHandler`](http://godoc.org/github.com/albrow/zoom/#ReplyHandler) as an
argument. A `ReplyHandler` is simply a function that will do something with the
reply from Redis. `ReplyHandler`'s are executed in order when you call `Exec`.

Right out of the box, Zoom exports a few useful `ReplyHandler`s. These include
handlers for the primitive types `int`, `string`, `bool`, and `float64`, as well
as handlers for scanning a reply into a `Model` or a slice of `Model`s. You can
also write your own custom `ReplyHandler`s if needed.


Queries
-------

### The Query Object

Zoom provides a useful abstraction for querying the database. You create queries by using the `NewQuery`
constructor, where you must pass in the name corresponding to the type of model you want to query. For now,
Zoom only supports queries on a single collection at a time.

You can add one or more query modifiers to the query, such as `Order`, `Limit`, and `Filter`. These methods
return the query itself, so you can chain them together. The first error (if any) that occurs due to invalid
arguments in the query modifiers will be remembered and returned when you attempt to run the query.

Finally, you run the query using a query finisher method, such as `Run` or `Count`. Queries feature delayed
execution, so nothing touches the database until you execute the query with a finisher method.

### Using Query Modifiers

You can chain a query object together with one or more different modifiers. Here's a list
of all the available modifiers:

- [`Order`](http://godoc.org/github.com/albrow/zoom/#Query.Order)
- [`Limit`](http://godoc.org/github.com/albrow/zoom/#Query.Limit)
- [`Offset`](http://godoc.org/github.com/albrow/zoom/#Query.Offset)
- [`Include`](http://godoc.org/github.com/albrow/zoom/#Query.Include)
- [`Exclude`](http://godoc.org/github.com/albrow/zoom/#Query.Exclude)
- [`Filter`](http://godoc.org/github.com/albrow/zoom/#Query.Filter)

You can run a query with one of the following query finishers:

- [`Run`](http://godoc.org/github.com/albrow/zoom/#Query.Run)
- [`IDs`](http://godoc.org/github.com/albrow/zoom/#Query.IDs)
- [`Count`](http://godoc.org/github.com/albrow/zoom/#Query.Count)
- [`RunOne`](http://godoc.org/github.com/albrow/zoom/#Query.RunOne)

Here's an example of a more complicated query using several modifiers:

``` go
people := []*Person{}
q := People.NewQuery().Order("-Name").Filter("Age >=", 25).Limit(10)
if err := q.Run(&people); err != nil {
	// handle error
}
```

Full documentation on the different modifiers and finishers is available on
[godoc.org](http://godoc.org/github.com/albrow/zoom/#Query).

### A Note About String Indexes

Because Redis does not allow you to use strings as scores for sorted sets, Zoom relies on a workaround
to store string indexes. It uses a sorted set where all the scores are 0 and each member has the following
format: `value\x00id`, where `\x00` is the NULL character. With the string indexes stored this way, Zoom
can issue the ZRANGEBYLEX command and related commands to filter models by their string values. As a consequence,
here are some caveats to keep in mind:

- Strings are sorted by ASCII value, exactly as they appear in an [ASCII table](http://www.asciitable.com/),
  not alphabetically. This can have surprising effects, for example 'Z' is considered less than 'a'.
- Indexed string values may not contain the NULL or DEL characters (the characters with ASCII codepoints
  of 0 and 127 respectively). Zoom uses NULL as a separator and DEL as a suffix for range queries.


More Information
----------------

### Persistence

Zoom is as persistent as the underlying Redis database. If you intend to use Redis as a permanent
datastore, it is recommended that you turn on both AOF and RDB persistence options and set `fsync` to
`everysec`. This will give you good performance while making data loss highly unlikely.

If you want greater protections against data loss, you can set `fsync` to `always`. This will hinder performance
but give you persistence guarantees
[very similar to SQL databases such as PostgreSQL](http://redis.io/topics/persistence#ok-so-what-should-i-use).

[Read more about Redis persistence](http://redis.io/topics/persistence)

### Atomicity

All methods and functions in Zoom that touch the database do so atomically. This is accomplished using
Redis transactions and Lua scripts when necessary. What this means is that Zoom will not
put Redis into an inconsistent state (e.g. where indexes to not match the rest of the data).

However, it should be noted that there is a caveat with Redis atomicity guarantees. If Redis crashes
in the middle of a transaction or script execution, it is possible that your AOF file can become
corrupted. If this happens, Redis will refuse to start until the AOF file is fixed. It is relatively
easy to fix the problem with the `redis-check-aof` tool, which will remove the partial transaction
from the AOF file.

If you intend to issue Redis commands directly or run custom scripts, it is highly recommended that
you also make everything atomic. If you do not, Zoom can no longer guarantee that its indexes are
consistent. For example, if you change the value of a field which is indexed, you should also
update the index for that field in the same transaction. The keys that Zoom uses for indexes
and models are provided via the [`ModelKey`](http://godoc.org/github.com/albrow/zoom/#Collection.ModelKey),
[`AllIndexKey`](http://godoc.org/github.com/albrow/zoom/#Collection.AllIndexKey), and
[`FieldIndexKey`](http://godoc.org/github.com/albrow/zoom/#Collection.FieldIndexKey) methods.

Read more about:
- [Redis persistence](http://redis.io/topics/persistence)
- [Redis scripts](http://redis.io/commands/eval)
- [Redis transactions](http://redis.io/topics/transactions)

### Concurrent Updates and Optimistic Locking

Zoom 0.18.0 introduced support for basic optimistic locking. You can use
optimistic locking to safely implement concurrent "read before write" updates.

Optimistic locking utilizes the `WATCH`, `MULTI`, and `EXEC` commands in Redis
and only works in the context of transactions. You can use the
[`Transaction.Watch`](https://godoc.org/github.com/albrow/zoom#Transaction.Watch)
method to watch a model for changes. If the model changes after you call `Watch`
but before you call `Exec`, the transaction will not be executed and instead
will return a
[`WatchError`](https://godoc.org/github.com/albrow/zoom#WatchError). You can
also use the `WatchKey` method, which functions exactly the same but operates on
keys instead of models.

To understand why optimistic locking is useful, consider the following code:

``` go
// likePost increments the number of likes for a post with the given id.
func likePost(postID string) error {
  // Find the Post with the given postID
  post := &Post{}
  if err := Posts.Find(postID, post); err != nil {
	 return err
  }
  // Increment the number of likes
  post.Likes += 1
  // Save the post
  if err := Posts.Save(post); err != nil {
	 return err
  }
}
```

The line `post.Likes += 1` is a "read before write" operation. That's because
the `+=` operator implicitly reads the current value of `post.Likes` and then
adds to it.

This can cause a bug if the function is called across multiple goroutines or
multiple machines concurrently, because the `Post` model can change in between
the time we retrieved it from the database with `Find` and saved it again with
`Save`.

You can use optimistic locking to avoid this problem. Here's the revised code:

```go
// likePost increments the number of likes for a post with the given id.
func likePost(postID string) error {
  // Start a new transaction and watch the post key for changes. It's important
  // to call Watch or WatchKey *before* finding the model.
  tx := pool.NewTransaction()
  if err := tx.WatchKey(Posts.ModelKey(postID)); err != nil {
    return err
  }
  // Find the Post with the given postID
  post := &Post{}
  if err := Posts.Find(postID, post); err != nil {
	 return err
  }
  // Increment the number of likes
  post.Likes += 1
  // Save the post in a transaction
  tx.Save(Posts, post)
  if err := tx.Exec(); err != nil {
  	 // If the post was modified by another goroutine or server, Exec will return
  	 // a WatchError. You could call likePost again to retry the operation.
    return err
  }
}
```

Optimistic locking is not appropriate for models which are frequently updated,
because you would almost always get a `WatchError`. In fact, it's called
"optimistic" locking because you are optimistically assuming that conflicts will
be rare. That's not always a safe assumption.

Don't forget that Zoom allows you to run Redis commands directly. This
particular problem might be best solved by the `HINCRBY` command.

```go
// likePost atomically increments the number of likes for a post with the given
// id and then returns the new number of likes.
func likePost(postID string) (int, error) {
	// Get the key which is used to store the post in Redis
	postKey := Posts.ModelKey(postID, post)
	// Start a new transaction
	tx := pool.NewTransaction()
	// Add a command to increment the number of Likes. The HINCRBY command returns
	// an integer which we will scan into numLikes.
	var numLikes int
	tx.Command(
		"HINCRBY",
		redis.Args{postKey, "Likes", 1},
		zoom.NewScanIntHandler(&numLikes),
	)
	if err := tx.Exec(); err != nil {
		return 0, err
	}
	return numLikes, nil
}
```

Finally, if optimistic locking is not appropriate and there is no built-in Redis
command that offers the functionality you need, Zoom also supports custom Lua
scripts via the
[`Transaction.Script`](https://godoc.org/github.com/albrow/zoom#Transaction.Script)
method. Redis is single-threaded and scripts are always executed atomically, so
you can perform complicated updates without worrying about other clients
changing the database.

Read more about:
- [Redis Commands](http://redis.io/commands)
- [Redigo](https://github.com/garyburd/redigo), the Redis Driver used by Zoom
- [`ReplyHandler`s provided by Zoom](https://godoc.org/github.com/albrow/zoom)
- [How Zoom works Under the Hood](https://github.com/albrow/zoom/wiki/Under-the-Hood)


Testing & Benchmarking
----------------------

### Running the Tests

To run the tests, make sure you're in the root directory for Zoom and run:

```
go test
```   

If everything passes, you should see something like:

```
ok    github.com/albrow/zoom  2.267s
```

If any of the tests fail, please [open an issue](https://github.com/albrow/zoom/issues/new) and
describe what happened.

By default, tests and benchmarks will run on localhost:6379 and use database #9. You can change the address,
network, and database used with flags. So to run on a unix socket at /tmp/redis.sock and use database #3,
you could use:

```
go test -network=unix -address=/tmp/redis.sock -database=3
```

### Running the Benchmarks

To run the benchmarks, make sure you're in the root directory for the project and run:

```
go test -run=none -bench .
```   

The `-run=none` flag is optional, and just tells the test runner to skip the tests and run only the benchmarks
(because no test function matches the pattern "none"). You can also use the same flags as above to change the
network, address, and database used.

You should see some runtimes for various operations. If you see an error or if the build fails, please
[open an issue](https://github.com/albrow/zoom/issues/new).

Here are the results from my laptop (2.8GHz quad-core i7 CPU, 16GB 1600MHz RAM) using a socket connection with Redis set
to append-only mode:

```
BenchmarkConnection-8                  	 5000000	       318 ns/op
BenchmarkPing-8                        	  100000	     15146 ns/op
BenchmarkSet-8                         	  100000	     18782 ns/op
BenchmarkGet-8                         	  100000	     15556 ns/op
BenchmarkSave-8                        	   50000	     29307 ns/op
BenchmarkSave100-8                     	    3000	    546427 ns/op
BenchmarkFind-8                        	   50000	     24767 ns/op
BenchmarkFind100-8                     	    5000	    374947 ns/op
BenchmarkFindAll100-8                  	    5000	    383919 ns/op
BenchmarkFindAll10000-8                	      30	  47267433 ns/op
BenchmarkDelete-8                      	   50000	     29902 ns/op
BenchmarkDelete100-8                   	    3000	    530866 ns/op
BenchmarkDeleteAll100-8                	    2000	    730934 ns/op
BenchmarkDeleteAll1000-8               	     200	   9185093 ns/op
BenchmarkCount100-8                    	  100000	     16411 ns/op
BenchmarkCount10000-8                  	  100000	     16454 ns/op
BenchmarkQueryFilterInt1From1-8        	   20000	     82152 ns/op
BenchmarkQueryFilterInt1From10-8       	   20000	     83816 ns/op
BenchmarkQueryFilterInt10From100-8     	   10000	    144206 ns/op
BenchmarkQueryFilterInt100From1000-8   	    2000	   1010463 ns/op
BenchmarkQueryFilterString1From1-8     	   20000	     87347 ns/op
BenchmarkQueryFilterString1From10-8    	   20000	     88031 ns/op
BenchmarkQueryFilterString10From100-8  	   10000	    158968 ns/op
BenchmarkQueryFilterString100From1000-8	    2000	   1088961 ns/op
BenchmarkQueryFilterBool1From1-8       	   20000	     82537 ns/op
BenchmarkQueryFilterBool1From10-8      	   20000	     84556 ns/op
BenchmarkQueryFilterBool10From100-8    	   10000	    149463 ns/op
BenchmarkQueryFilterBool100From1000-8  	    2000	   1017342 ns/op
BenchmarkQueryOrderInt100-8            	    3000	    386156 ns/op
BenchmarkQueryOrderInt10000-8          	      30	  50011375 ns/op
BenchmarkQueryOrderString100-8         	    2000	   1004530 ns/op
BenchmarkQueryOrderString10000-8       	      20	  77855970 ns/op
BenchmarkQueryOrderBool100-8           	    3000	    387056 ns/op
BenchmarkQueryOrderBool10000-8         	      30	  49116863 ns/op
BenchmarkComplexQuery-8                	   20000	     84614 ns/op
```

The results of these benchmarks can vary widely from system to system, and so the benchmarks
here are really only useful for comparing across versions of Zoom, and for identifying possible
performance regressions or improvements during development. You should run your own benchmarks that
are closer to your use case to get a real sense of how Zoom will perform for you. High performance
is one of the top priorities for this project.


Contributing
------------

See [CONTRIBUTING.md](https://github.com/albrow/zoom/blob/master/CONTRIBUTING.md).


Example Usage
-------------

[albrow/people](https://github.com/albrow/people) is an example HTTP/JSON API
which uses the latest version of Zoom. It is a simple example that doesn't use
all of Zoom's features, but should be good enough for understanding how Zoom can
work in a real application.


License
-------

Zoom is licensed under the MIT License. See the LICENSE file for more information.
