Zoom
====

A blazing-fast, lightweight ORM-ish library for go and redis.

Why "ORM-ish" instead of just ORM. Go doesn't really have objects in the traditional sense.
Perhaps it would be more accurate to call it Struct Relational Mapping, but no one would really
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
For now, Zoom assumes that you will be accepting socket connections on `/tmp/redis.sock`. Zoom will use
database 7. Soon you will have the ability to configure the connection if you want to use a different database
id or a tcp connection, but for now these are hardcoded.

To install Zoom itself:

    go get github.com/stephenalexbrowne/zoom
    
This will pull the current master branch, which is (most likely) stable but is quickly changing.

Usage
-----

### Setup

First, add github.com/stephenalexbrowne/zoom to your import statement:

``` go
import (
    ...
    github.com/stephenalexbrowne/zoom
)
```

In order to save your struct using Zoom, you need to add an embedded struct attribute like so:

``` go
type Person struct {
    Name String
    *zoom.Model
}
```

The *zoom.Model embedded attribute automatically gives you an Id field. You do not need to add an Id field
to your struct.

*Important:* In your constructor, you must initialize the embedded *zoom.Model attribute. Something like this:

``` go
func NewPerson(name string) *Person {
    return &Person{
        Name:  name,
        Model: new(zoom.Model), // don't forget this!
    }
}
```

You must also call zoom.Register() so that Zoom knows what name to assign to the *Person type. Somewhere in
your initialization sequence (e.g. in the main() method) put the following:

``` go
zoom.Init() // later, this will require config options

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
    
Example Usage
-------------

The [zoom_example repository](https://github.com/stephenalexbrowne/zoom_example) is an up-to-date example
of how to use Zoom in a json/rest application. Use it as a reference if anything above is not clear. Formal
documentation is on my todo list.

