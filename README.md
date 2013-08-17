Zoom
====

Version: 0.0.1

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
[Check the benchmarks](#running-the-benchmarks).


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

### Saving to Redis

To persistently save a Person model to the databse, you would simply call zoom.Save()

``` go
p := NewPerson("Alice")
if err := zoom.Save(p); err != nil {
    // handle error
}
```

### Retreiving from Redis

Zoom will automatically assign a random, unique id to each saved model. To retrieve a model by id,
you must use the same string name you used in zoom.Register. The return type
is interface{}, so you may need to cast to *Person using a type assertion.

``` go
result, err := zoom.FindById("person", "your-person-id")
if err != nil {
    // handle error
}

// type assert
person, ok := result.(*Person)
if !ok {
    // handle !ok
}
```
    
To retrieve a list of all persons use zoom.FindAll(). Like FindById() the return type is []interface{}.
If you want an array or slice of *Person, you need to type assert each element individually.

``` go
results, err := zoom.FindAll("person")
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

### One-to-One Relations

Relations in Zoom are dead simple. There are no special return types or methods for using relations.
What you put in is what you get out. For these examples we're going to introduce two new struct types:

``` go
// The PetOwner struct
type PetOwner struct {
	Name  string
	Pet   *Pet    // *Pet should also be a registered type
	*zoom.Model
}

// A convenient constructor for the PetOwner struct
func NewPetOwner(name string) *PetOwner {
	return &PetOwner{
		Name:  name,
		Model: new(zoom.Model),
	}
}

// The Pet struct
type Pet struct {
	Name   string
	*zoom.Model
}

// A convenient constructor for the Pet struct
func NewPet(name string) *Pet {
	return &Pet{
		Name:  name,
		Model: new(zoom.Model),
	}
}
```

Assuming you've registered both the *PetOwner and *Pet types, Zoom will automatically set up a relation
when you save a PetOwner with a non-nil Pet.

``` go
// create a new PetOwner and a Pet
owner := NewPetOwner("Bob")
pet := NewPet("Spot")

// set the PetOwner's pet
owner.Pet = pet

// save it
if err := zoom.Save(owner); err != nil {
	// handle error
}
```

Behind the scenes, Zoom creates a seperate database entry for Pet and assigns it an Id. Now if you retrieve
the PetOwner by it's id, the Pet attribute will persist as well.

For now, Zoom does not support reflexivity of one-to-one relations. So if you want pet ownership to be
bidirectional (i.e. if you want an owner to know about its pet **and** a pet to know about its owner),
you would have to manually set up both relations.

``` go
reply, err := zoom.FindById("petOwner", "the_id_of_above_pet_owner")
if err != nil {
	// handle error
}

// don't forget to type assert
ownerCopy, ok := reply.(*PetOwner)
if !ok {
	// handle this
}

// the Pet attribute is still set
fmt.Println(ownerCopy.Pet.Name)

// output:
//	Spot
```

The Pet is stored in its own database entry, so you could retreive the pet named "Spot" directly by 
it's id (if you know it). You can also examine the list of pets to see that it is there.

``` go
// this would work
reply, err := zoom.FindById("pet", "the_id_of_above_pet")

// this would work too
results, err := zoom.FindAll("pet")
// results would contain the one Pet named Spot
```


### One-to-Many Relations

One-to-many relations work similarly. This time we're going to use two new struct types in the examples.

``` go
// The Parent struct
type Parent struct {
	Name     string
	Children []*Child  // *Child should also be a registered type
	*zoom.Model
}

// A convenient constructor for the Parent struct
func NewParent(name string) *Parent {
	return &Parent{
		Name:  name,
		Model: new(zoom.Model),
	}
}

// The Child struct
type Child struct {
	Name   string
	*zoom.Model
}

// A convenient constructor for the Child struct
func NewChild(name string) *Child {
	return &Child{
		Name:  name,
		Model: new(zoom.Model),
	}
}
```

Assuming you register both the *Parent and *Child types, Zoom will automatically set up a relation
for you when you save a Parent with non-nil Children. Here's an example:

``` go
// create a parent and two children
parent := NewParent("Christine")
child1 := NewChild("Derick")
child2 := NewChild("Elise")

// assign the children to the parent
parent.Children = append(parent.Children, child1, child2)

// save the parent
if err := zoom.Save(parent); err != nil {
	// handle error
}
```

When the above code is run, Zoom will create a database entry for each child, and give them a unique id.

Again, Zoom does not support reflexivity. So if you wanted a Child to know about its Parent, you would have
to set up and manage the relation manually. This might change in the future.

Now when you retrieve a parent by id, it's Children attribute will automatically be populated. So getting
the children again is straight forward.

``` go
reply, err := zoom.FindById("parent", "the_id_of_above_parent")
if err != nil {
	// handle error
}

// don't forget to type assert
parentCopy, ok := reply.(*Parent)
if !ok {
	// handle this
}

// now you can access the children normally
for _, child := range parentCopy.Children {
	fmt.Println(child.Name)
}

// output:
//	Derick
//	Elise

```

Since each child is its own database entry, you could also access the children directly or get a list
of all children.

``` go
// this would work
reply, err := zoom.FindById("child", "the_id_of_an_above_child")

// this would work too
results, err := zoom.FindAll("child")
// results would contain both children

```

### Many-to-Many Relations

There is nothing special about many-to-many relations. They are basically made up of multiple one-to-many
relations.

Here's an example:

``` go
// The Friend struct
type Friend struct {
	Name    string
	Friends []*Friend
	*zoom.Model
}

// A convenient constructor for the Friend struct
func NewFriend(name string) *Friend {
	return &Friend{
		Name:  name,
		Model: new(zoom.Model),
	}
}
```

Each Friend struct holds a list of his/her friends. Assuming a two-way friend relationship,
if Joe is friends with Amy, there would be two entries: Amy's id is in Joe's list of friends and Joe's id is in
Amy's list of friends. This redundancy makes queries faster at the expense of higher memory usage. Since there are
no joins needed to lookup Joe's friends, we can get them from the database quickly. The latency scales
linearly with the number of Joe's friends, regardless of the total number of Friend entries or the total number
of Friend-to-Friend relations.

Here's how you would save some friends in the database:


``` go
// create 5 people
fred := NewFriend("Fred")
george := NewFriend("George")
hellen := NewFriend("Hellen")
ilene := NewFriend("Ilene")
jim := NewFriend("Jim")

// Fred is friends with George, Hellen, and Jim
fred.Friends = append(fred.Friends, george, hellen, jim)

// George is friends with Fred, Hellen, and Ilene
george.Friends = append(george.Friends, fred, hellen, ilene)

// save both Fred and George
if err := zoom.Save(fred); err != nil {
	c.Error(err)
}
if err := zoom.Save(george); err != nil {
	c.Error(err)
}
```

Recall that Zoom does not support reflexivity of many-to-many relations. So if you want friendships to be bidirectional,
you would have to manually add each person to the list of the other's friends. This might change in the future.

Also note that in the above example, Zoom will create separate database entries for Hellen, Ilene, and Jim
because they are related to Fred and George, even though we did not call Save() explicitly.

Retrieving many-to-many relations works exactly the same as one-to-many. When you get a Friend struct from
the database using FindById or FindAll, the Friends array is auto-filled. You get out whatever you put in.

``` go
// retrieve fred from the database
result, err := zoom.FindById("friend", fred.Id)
if err != nil {
	// handle err
}

// type assert to *Friend
fredCopy, ok := result.(*Friend)
if !ok {
	// handle this
}

for _, friend := range fred.Friends {
	fmt.Println(friend.Name)
}

// output:
//	George
// 	Hellen
//	Jim
```




Testing & Benchmarking
----------------------

**Important:** Before running any tests or benchmarks, make sure that you have redis running and accepting
connections on a unix socket at /tmp/unix.sock. To test if your redis server is properly set up, you can run:

    redis-cli -s /tmp/redis.sock -c ping
    
If you receive PONG in response, then you are good to go. If anything else happens, redis is not setup
properly. Check out the [official redis docs](http://redis.io/topics/config) for help. You might also find
the [redis quickstart guide](http://redis.io/topics/quickstart) helpful, especially the bottom sections.

All the tests and benchmarks will use database #9. If database #9 is non-empty, they will will throw and
error and not run. (so as to not corrupt your data). Database #9 is flushed at the end of every test/benchmark.

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


TODO
----

In no particular order, here's what I'm working on:

- Use transactions where possible to increase performance and robustness (a la multi/exec)
- Implement sorting
- Add CreatedAt and UpdatedAt attributes to zoom.Model
- Be able to save arrays embedded in structs
- Implement saving arbitrary embedded structs (even if not registered)
- Write good, formal documentation
- Re-implement low-level pub/sub (currently missing entirely)
- Implement high-level watching for record changes


LICENSE
-------

Zoom is licensed under the MIT License. See the LICENSE file for more information.

The redis driver for Zoom is based on https://github.com/garyburd/redigo and is licensed under
the Apache License, Version 2.0. See redis/NOTICE for more information. Thanks Gary!
