Zoom
====

[![GoDoc](https://godoc.org/github.com/albrow/zoom?status.svg)](https://godoc.org/github.com/albrow/zoom)

Version: X.X.X

A blazing-fast datastore and querying engine for Go built on Redis.

Requires Redis version >= 2.8.9 and Go version >= 1.2.

Full documentation is available on
[godoc.org](http://godoc.org/github.com/albrow/zoom).

**WARNING:** this isn't done yet and may change significantly before the official release. There is no
promise of backwards compatibility until version 1.0. I do not advise using Zoom for production or
mission-critical applications. Feedback and pull requests are welcome :)

Table of Contents
-----------------

- [Usefulness](#usefulness)
- [Installation](#installation)
- [Initialization](#initialization)
- [Models](#models)
- [Transactions](#transactions)
- [Queries](#queries)
- [More Information](#more-information)
- [Testing & Benchmarking](#testing--benchmarking)
- [Example Usage](#example-usage)
- [TODO](#todo)
- [License](#license)


Usefulness
----------

Zoom may be a good fit for you if:

1. You require high performance or low latency and don't want to write complicated database components from scratch
2. You are using Redis and want the ability to do queries
3. You are using Redis with some custom types and want to easily do conversions to store and retrieve those types

Zoom allows you to:

- Persistently save models of any type
- Retrieve models from the database
- Perform basic queries

Zoom consciously makes the trade off of using more memory in order to increase performance.
Zoom stores all data in memory at all times, so if your machine runs out of memory, Zoom will
either crash or start using swap space (resulting in huge performance penalties). 
Zoom does not do sharding and does not support Redis Cluster (but might in the future), so be
aware that memory could be a hard constraint for larger applications.

Zoom is a high-level library and abstracts away more complicated aspects of the Redis API. For example,
it manages its own connection pool, performs atomic transactions with MULTI/EXEC or lua scripts when necessary,
automatically converts structs to and from a format suitable for the database, and manages query indexes using
sorted sets. If needed, you can still execute Redis commands directly.

If you want to use advanced/complicated SQL queries, Zoom is not for you. For example, Zoom
currently lacks an equivalent of the SQL keywords `IN` and `OR`. Although support for more
types of queries may be added in the future, it is not a high priority.


Installation
------------

Zoom is powered by Redis and needs to connect to a Redis database. You can install Redis on the same
machine that Zoom runs on, connect to a remote database, or even use a Redis-as-a-service provider such
as Redis To Go, RedisLabs, or Amazon Elasticache.

If you need to install Redis, see the [installation instructions](http://redis.io/download) on the official
Redis website.

To install Zoom itself:

    go get github.com/albrow/zoom
    
This will pull the current master branch, which is (most likely) working but is quickly changing.


Initialization
--------------

First, add github.com/albrow/zoom to your import statement:

``` go
import (
    // ...
    github.com/albrow/zoom
)
```

Then, call zoom.Init somewhere in your app initialization code, e.g. in the main function. You must
also call zoom.Close when your application exits, so it's a good idea to use defer.

``` go
func main() {
    // ...
    zoom.Init(nil)
    defer zoom.Close()
    // ...
}
```

The Init function takes a *zoom.Configuration struct as an argument. Here's a list of options and their
defaults:

``` go
type Configuration struct {
   // Address to connect to. Default: "localhost:6379"
  Address string
  // Network to use. Default: "tcp"
  Network string
  // Database id to use (using SELECT). Default: 0
  Database int
  // Password for a password-protected Redis database. If not empty,
  // every connection will use the AUTH command during initialization
  // to authenticate with the database. Default: ""
  Password string
}
```

If you pass in nil to Init, Zoom will use all the default values. Any fields in the Configuration struct
that are empty (i.e. an empty string or 0) will fall back to their default values, so you only need to
provide a Configuration struct with the fields you want to change.


Models
------

### What is a Model?

Models in Zoom are just structs which implement the zoom.Model interface:

``` go
type Model interface {
  Id() string
  SetId(string)
}
```

To clarify, all you have to do to implement the Model interface is add getters and setters
for a unique Id property. If you want, you can embed zoom.DefaultData to give your model all the
required methods.

A struct definition serves as a sort of schema for your model. Here's an example of a Person model:

``` go
type Person struct {
    Name string
    Age  int
    zoom.DefaultData
}
```

Because of the way Zoom uses reflection, all the fields you want to save need to be public.

You must also call zoom.Register on each Model type in your application. You only need to do this
once per type.

``` go
// register the *Person type and assign the corresponding *ModelType to the variable named Persons
Persons, err := zoom.Register(&Person{})
if err != nil {
    // handle error
}
```

Register returns a *ModelType, which is a reference to a registered model type and has methods for
saving, deleting, and querying models of that type. Convention is to name the *ModelType the plural
of the corresponding registered type, but it's just a variable so you can name it whatever you want.


### Saving Models

To persistently save a Person model to the database, you could call Persons.Save. Recall that in this
example, Persons is just the name we gave to the *ModelType which corresponds to the type *Person.

``` go
p := &Person{Name: "Alice", Age: 27}
if err := Persons.Save(p); err != nil {
    // handle error
}
```

When you call Save, Zoom converts all the fields of the model into a format suitable for Redis and stores them
as a Redis hash. Struct fields can be any custom or built-in type, but cannot be functions or recursive data
structures. If the model you are saving does not already have an id, Zoom will mutate the model by generating and
assigning one via the SetId method. Ids assigned by Zoom have two components: the current unix time with
millisecond precision and a randomly generated 16-character string. When Ids are generated this way, collisions
are still possible, but they are highly, highly unlikely.

### Finding a Single Model

To retrieve a model by id, use the Find method:

``` go
p := &Person{}
if err := Persons.Find("a_valid_person_id", p); err != nil {
    // handle error
}
```

The second argument to Find must be a pointer to a Model, with a type corresponding to the *ModelType. In this case,
we passed in *Person since that is the type that corresponds to our *ModelType Persons. Find will mutate p by setting
all its fields. Using Find in this way allows the caller to maintain type safety and avoid type casting. If Zoom couldn't
find a model of type *Person with the given id, it will return a
[ModelNotFoundError](http://godoc.org/github.com/albrow/zoom/#ModelNotFoundError).

### Finding All Models

To find all models of a given type, use the FindAll method:

``` go
persons := []*Person{}
if err := Persons.FindAll(&persons); err != nil {
    // handle error
}
```

FindAll expects a pointer to a slice of some registered type. It grows or shrinks the slice as needed, filling in all
the fields of the elements inside of the slice. So the result of the call is that persons will be a slice of all models
in the database with the type *Person.

### Deleting Models

To delete a model, use the Delete method:

``` go
// ok will be true iff a model with the given id existed and was deleted
if ok, err := Persons.Delete("a_valid_person_id"); err != nil {
   // handle err
}
```

Delete expects a valid id as an argument, and will attempt to delete the model with the given id. If there was no model
with the given type and id, the first return value will be false.

You can also delete all models of a given type with the DeleteAll method:

``` go
numDeleted, err := Persons.DeleteAll()
if err != nil {
  // handle error
}
```

DeleteAll will return the number of models that existed and were successfully deleted.

### Counting the Number of Models

You can get the number of models for a given type using the Count method:

``` go
count, err := Persons.Count()
if err != nil {
  // handle err
}
```


Transactions
------------

Zoom exposes a Transaction API which you can use to run multiple commands efficiently and atomically. Under the hood,
Zoom uses a single [Redis transaction](http://redis.io/topics/transactions) to perform all the commands in a single
round trip. Transactions feature delayed execution, so nothing touches the database until you call Exec. A transaction
also remembers its errors to make error handling easier on the caller. Any errors that occur will be returned when you
call Exec.

Here's an example of how to save two models and get the number of *Person models in a single transaction.

``` go
numPersons := 0
t := NewTransaction()
t.Save(Persons, &Person{Name: "Foo"})
t.Save(Persons, &Person{Name: "Bar"})
// Count expects a pointer to an integer, which it will change the value of
// when the transaction is executed.
t.Count(Persons, &numPersons)
if err := t.Exec(); err != nil {
  // handle error
}
// numPersons will now equal the number of *Person models in the database
fmt.Println(numPersons)
```

You can also execute custom Redis commands or run lua scripts with the
[Command](http://godoc.org/github.com/albrow/zoom/#Transaction.Command) and
[Script](http://godoc.org/github.com/albrow/zoom/#Transaction.Command) methods. Both methods expect a
[ReplyHandler](http://godoc.org/github.com/albrow/zoom/#ReplyHandler) as an argument. A ReplyHandler is
simply a function that will do something with the reply from Redis corresponding to the script or command
that was run. ReplyHandler's are executed in order when you call Exec.


Queries
-------

### The Query Object

Zoom provides a useful abstraction for querying the database. You create queries by using the NewQuery
constructor, where you must pass in the name corresponding to the type of model you want to query. For now,
Zoom only supports queries on a single type of model at a time.

You can add one or more query modifiers to the query, such as Order, Limit, and Filter. These methods return
the query itself, so you can chain them together. Any errors due to invalid arguments in the query modifiers
will be remembered and returned when you attempt to run the query.

Finally, you run the query using a query finisher method, such as Run or Count. Queries feature delayed
execution, so nothing touches the database until you execute the query with a finisher method.

### Using Query Modifiers

You can chain a query object together with one or more different modifiers. Here's a list
of all the available modifiers:

- Order
- Limit
- Offset
- Include
- Exclude
- Filter

You can run a query with one of the following query finishers:

- Run
- Ids
- Count
- RunOne

Here's an example of a more complicated query using several modifiers:

``` go
persons := []*Person{}
q := Persons.NewQuery().Order("-Name").Filter("Age >=", 25).Limit(10)
if err := q.Run(&persons); err != nil {
   // handle error
}
```

Full documentation on the different modifiers and finishers is available on
[godoc.org](http://godoc.org/github.com/albrow/zoom/#Query).

### A Note About String Indexes

Because Redis does not allow you to use strings as scores for sorted sets, Zoom relies on a workaround
to store string indexes. It uses a sorted set where all the scores are 0 and each member has the following
format: `value\0id`, where `\0` is the NULL character. With the string indexes stored this way, Zoom
can issue the ZRANGEBYLEX command and related commands to filter models by their string values. As a consequence,
here are some caveats to keep in mind:

- Strings are sorted by ASCII value, exactly as they appear in an [ASCII table](http://www.asciitable.com/),
  not alphabetically. This can have surprising effects, for example 'Z' is considered less than 'a'.
- Indexed string values may not contain the NULL or DEL characters (the characters with ASCII codepoints
  of 0 and 127 respectively). Zoom uses NULL as a separator and DEL as a suffix for range queries.


More Information
----------------

### Persistence

Zoom is as persistent as the underlying Redis database is. If you intend to use Redis as a permanent
datastore, it is recommended that you turn on both AOF and RDB persistence options and set fsync to
everysec. This will give you good performance while making data loss highly unlikely.

If you want greater protections against data loss, you can set fsync to always. This will hinder performance
but give you persistence guarantees
[very similar to SQL databases such as PostgreSQL](http://redis.io/topics/persistence#ok-so-what-should-i-use).

[Read more about Redis persistence](http://redis.io/topics/persistence)

### Atomicity

All methods and functions in Zoom that touch the database do so atomically. This is accomplished using
Redis transactions and lua scripts when necessary. What this means is that Zoom will not
put Redis into an inconsistent state (e.g. where indexes to not match the rest of the data).

However, it should be noted that there is a caveat with Redis atomicity guarantees. If Redis crashes
in the middle of a transaction or script execution, it is possible that your AOF file can become
corrupted. If this happens, Redis will refuse to start until the AOF file is fixed. It is relatively
easy to fix the problem with the redis-check-aof tool, which will remove the partial transaction
from the AOF file.

If you intend to issue custom Redis commands or run custom scripts, it is highly recommended that
you also make everything atomic. If you do not, Zoom can no longer guarantee that its indexes are
consistent. For example, if you change the value of a field which is indexed, you should also
update the index for that field in the same transaction. The keys that Zoom uses for indexes
and models are provided via the [ModelKey](http://godoc.org/github.com/albrow/zoom/#ModelType.ModelKey),
[AllIndexKey](http://godoc.org/github.com/albrow/zoom/#ModelType.AllIndexKey), and
[FieldIndexKey](http://godoc.org/github.com/albrow/zoom/#ModelType.FieldIndexKey) methods.

Read more about:
- [Redis persistence](http://redis.io/topics/persistence)
- [Redis scripts](http://redis.io/commands/eval)
- [Redis transactions](http://redis.io/topics/transactions)

### Thread-Safe Updates

Currently, Zoom does not support thread-safe or cross-machine updates on models. Consider the
following code:

``` go
func likePost(postId string) error {
  // Find the Post with the given postId
  post := &Post{}
  if err := Posts.Find(postId); err != nil {
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

This can cause a bug if the function is called across multiple threads or multiple machines
concurrently, because the Post model can change in between the time we retrieved it from the
database with Find and saved it again with Save. Future versions of Zoom will provide optimistic
locking or other means to avoid these kinds of errors. In the meantime, you could fix this code
by using an HINCRBY command directly like so:

``` go
func likePost(postId string) error {
  // modelKey is the key of the main hash for the model, which
  // stores the struct fields as hash fields in Redis.
  modelKey, err := Posts.ModelKey(postId)
  if err != nil {
    return err
  }
  conn := zoom.NewConn()
  defer conn.Close()
  if _, err := conn.Do("HINCRBY", modelKey, 1); err != nil {
    return err
  }
}
```

You could also use a lua script for more complicated thread-safe updates.


Testing & Benchmarking
----------------------

### Running the Tests:

To run the tests, make sure you're in the root directory for Zoom and run:

```
go test .
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
go test . -network=unix -address=/tmp/redis.sock -database=3
```

### Running the Benchmarks:

To run the benchmarks, make sure you're in the root directory for the project and run:

```
go test . -run=none -bench .
```   

The `-run=none` flag is optional, and just tells the test runner to skip the tests and run only the benchmarks
(because no test function matches the pattern "none"). You can also use the same flags as above to change the
network, address, and database used.

You should see some runtimes for various operations. If you see an error or if the build fails, please
[open an issue](https://github.com/albrow/zoom/issues/new).

Here are the results from my laptop (2.3GHz quad-core i7, 8GB RAM) using a socket connection with Redis set
to append-only mode:

```
BenchmarkConnection  2000000         656 ns/op
BenchmarkPing          50000       26627 ns/op
BenchmarkSet           50000       36452 ns/op
BenchmarkGet           50000       27864 ns/op
BenchmarkSave          20000       58962 ns/op
BenchmarkSave100        2000      960483 ns/op
BenchmarkFind          30000       43054 ns/op
BenchmarkFind100        3000      562743 ns/op
BenchmarkFindAll100     2000      665035 ns/op
BenchmarkFindAll10000     20    68657190 ns/op
BenchmarkDelete        20000       61379 ns/op
BenchmarkDelete100      2000     1031886 ns/op
BenchmarkDeleteAll100   2000      968367 ns/op
BenchmarkDeleteAll1000   100    11857145 ns/op
BenchmarkCount100      50000       28340 ns/op
BenchmarkCount10000    50000       29746 ns/op
BenchmarkQueryFilterInt1From1         10000      149719 ns/op
BenchmarkQueryFilterInt1From10        10000      148245 ns/op
BenchmarkQueryFilterInt10From100       5000      264959 ns/op
BenchmarkQueryFilterInt100From1000     1000     1654756 ns/op
BenchmarkQueryFilterString1From1      10000      152185 ns/op
BenchmarkQueryFilterString1From10     10000      154507 ns/op
BenchmarkQueryFilterString10From100    5000      287958 ns/op
BenchmarkQueryFilterString100From1000  1000     1862549 ns/op
BenchmarkQueryFilterBool1From1        10000      146349 ns/op
BenchmarkQueryFilterBool1From10       10000      147950 ns/op
BenchmarkQueryFilterBool10From100      5000      276740 ns/op
BenchmarkQueryFilterBool100From1000    1000     1641239 ns/op
BenchmarkQueryOrderInt100       2000      681141 ns/op
BenchmarkQueryOrderInt10000       20    72602768 ns/op
BenchmarkQueryOrderString100    1000     1662290 ns/op
BenchmarkQueryOrderString10000    10   118660109 ns/op
BenchmarkQueryOrderBool100      2000      681808 ns/op
BenchmarkQueryOrderBool10000      20    71249344 ns/op
BenchmarkComplexQuery          10000      142476 ns/op
```

The results of these benchmarks can vary widely from system to system, and so the benchmarks
here are really only useful for comparing accross versions of Zoom, and for identifying possible
performance regressions during development. You should run your own benchmarks that are closer to
your use case to get a real sense of how Zoom will perform for you. High performance is one of the
top priorities for this project, because without that you are better off using an ORM designed for
SQL databases.


Example Usage
-------------

I have built an [example json/rest application](https://github.com/albrow/peeps-negroni)
which uses the latest version of Zoom. It is a simple example that doesn't use all of
Zoom's features, but should be good enough for understanding how zoom can work in a
real application.


TODO
----

Ordered generally by priority, here's what I'm working on:

- Add godoc compatible examples in the test files
- Support callbacks (BeforeSave, AfterSave, BeforeDelete, AfterDelete, etc.)
- Implement high-level watching for record changes
- Implement thread-safe updates across different application servers (probably optimistic locking)
- Write a basic migration tool
- Support AND and OR operators on Filters

If you have an idea or suggestion for a feature, please [open an issue](https://github.com/albrow/zoom/issues/new)
and describe it.


License
-------

Zoom is licensed under the MIT License. See the LICENSE file for more information.
