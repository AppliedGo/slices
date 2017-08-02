/*
<!--
Copyright (c) 2017 Christoph Berger. Some rights reserved.

Use of the text in this file is governed by a Creative Commons Attribution Non-Commercial
Share-Alike License that can be found in the LICENSE.txt file.

Use of the code in this file is governed by a BSD 3-clause license that can be found
in the LICENSE.txt file.

The source code contained in this file may import third-party source code
whose licenses are provided in the respective license files.
-->

<!--
NOTE: The comments in this file are NOT godoc compliant. This is not an oversight.

Comments and code in this file are used for describing and explaining a particular topic to the reader. While this file is a syntactically valid Go source file, its main purpose is to get converted into a blog article. The comments were created for learning and not for code documentation.
-->

+++
title = "Go slices are not dynamic arrays"
description = "Go slices are based on a smart concept that does not like being ignored"
author = "Christoph Berger"
email = "chris@appliedgo.net"
date = "2017-08-01"
draft = "true"
domains = ["Patterns and Paradigms"]
tags = ["slice", "append", "split", "memory management", "gotcha"]
categories = ["Background"]
+++

Go's slices are cleverly designed. They provide the look-and-feel of truly dynamic arrays while being optimized for performance. However, not being aware of the slice mechanisms can bring you into trouble.

<!--more-->

The concept of slices in Go is really a clever one. A slice represents a flexible-length array-like data type while providing full control over memory allocations.

But first things first: What are slices, and how do they work?

## The slice concept in Go

In Go, arrays have a fixed size. The size is even part of the definition of an array, so the two arrays `[10]int` and `[20]int` are not just two `int` arrays of different size but are in fact different types.

Slices add a dynamic layer on top of arrays. Creating a slice from an array  neither allocates new memory nor copies anything. A slice is nothing but a "window" to some part of the array. Technically, a slice can be seen as a struct with a pointer to the array element where the slice starts, and two ints describing length and capacity.

This means that typical slice manipulations are cheap. Creating a slice, expanding it (as far as the available capacity permits), moving it back and forth on the underlying array--all that requires nothing more than changing the pointer value and/or one or both of the two int values. The data location does not change.

This also means that two slices created from the same array can overlap, and after assigning a slice to a new slice variable, both variables now share the same memory cells. Changing one item in one of the slices also change the same item in the other slice. If you want to create a true copy of a slice, create a new slice and call the built-in function `copy()`.

All of this is based on simple and consistent mechanisms.

The problems arise when not being aware of these mechanisms.


## Some slice functions work in place

In the spirit of slices as efficient "dynamic windows" on static arrays, functions that receive and return slices may apply their operations to the original slice.

As an example, `bytes.Split()` takes a slice and a separator, splits the slice by the separator, and returns a slice of byte slices.

But: All the byte slices returned by `Split()` still point to the same underlying array as the original slice. This may--no, this *will* come unexpected to those who know split functions from other languages that happily sacrifice efficiency for the convenience of allocate-and-copy semantics.

Code that ignores the fact that the result of `Split()` still points to the original data may cause data corruption in a way that neither the compiler nor the runtime can detect as being wrong.


## append() adds convenience--and magic

`append()` adds new elements to the end of a slice, thus expanding the slice. This is not different to re-slicing a slice--until the slice capacity is reached. Then `append()` takes the liberty to allocate a new array of sufficient length, copy over the original slice, and appending the new values to this new slice.

Now any slices that previously shared some or all elements with the re-allocated slice suddenly do not share any data with that slice anymore.

So `append()` can cause two different results, depending on whether the original array has sufficient room for expanding the slice, or whether allocating a new array is required.


And there is a second characteristic of `append()`. It receives the slice parameter *by value* (remember that all parameters are passed by value in Go), which means that the slice header is copied over into `append`'s body. Any change to the slice header--length, capacity, or the location in case of a new array--is therefore only stored in the local copy of the slice header. For this reason, `append()` returns the modified slice header to the caller, to allow updating the original slice header with any changes to location, length, or capacity.

But this does not mean that the return value is always used as intended. Developers who are not aware of `append()`'s semantics could be tempted to assign the result of `append()` to an entirely different variable, for example:

```
s1 := []int{1, 2, 3, 4}
s2 := append(s1, 5, 6, 7, 8)
```

Now the original slice header is not updated with the new length and perhaps also a new capacity, and a new location in case `append()` had to allocate a new array.


## A case study
*/

//
package main

import (
	"bytes"
	"fmt"
)

func splitDemo() {
	fmt.Println("Split demo")
	// bytes.Split splits in place.
	a := []byte("a,b,c")
	b := bytes.Split(a, []byte(","))
	fmt.Printf("b: %q\n", b)

	// b's byte slices use a's underlying array.
	fmt.Println("Setting a[0] to '*'")
	a[0] = byte('*')
	fmt.Printf("b: %q\n", b)
}

func appendDemo() {
	fmt.Println("\nAppend demo")
	s1 := []int{1, 2}
	fmt.Printf("%p: %[1]v\n", s1)
	s1 = append(s1, 4)
	fmt.Printf("%p: %[1]v\n", s1)
}

func alwaysCopy() {
	fmt.Println("append and always copy")
	s1 := []int{1, 2, 3, 4}
	// Create a new slice with sufficient len (for copy) and cap (for append).
	s2 := make([]int, 4, 8)
	// Destination is always the first parameter, analogous to Fprintf, http.HandleFunc, etc.
	copy(s2, s1)
	s2 = append(s2, 5, 6, 7, 8)
	fmt.Printf("%p: %[1]v\n", s1)
	fmt.Printf("%p: %[1]v\n", s2)
}

func neverCopy() {
	fmt.Println("append but never copy")
	// Create a new slice with len=4 and cap=8.
	s1 := make([]int, 4, 8)
	// Fill the slice with initial data.
	copy(s1, []int{1, 2, 3, 4})
	fmt.Printf("%p: %[1]v\n", s1)
	// Append without copying.

	fmt.Printf("%p: %[1]v\n", s1)
}

func main() {
	splitDemo()
	appendDemo()
	alwaysCopy()
	neverCopy()
}

/*
## How to get and run the code

Step 1: `go get` the code. Note the `-d` flag that prevents auto-installing
the binary into `$GOPATH/bin`.

    go get -d github.com/appliedgo/TODO:

Step 2: `cd` to the source code directory.

    cd $GOPATH/src/github.com/appliedgo/TODO:

Step 3. Run the binary.

    go run TODO:.go


## Conclusions

### Remember that append() may or may not allocate a new slice.

In many cases, this is absolutely ok, as a single slice does not care if it gets relocated. Only when two or more slices interact, the behavior of `append()` can lead to unexpected results.

**To avoid ambiguous results, use the correct techniques to ensure the desired outcome:**

* If you absolutely want to avoid allocation and copying, use a large underlying array, re-slice your slice, and strictly avoid `append()`.

* If you absolutely need copy semantics, create a destination slice of sufficient size, and use the built-in `copy()` function.


### Never assume exclusive ownership of a slice that you did not create.

Any function that returns a slice may return a *shared* slice. `bytes.Split` splits a slice in-place, `append()` returns a slice header that still might point to the slice that it received before.

Hence if you receive a slice from a function, keep in mind that other code may still modify that slice. Again, `copy()` is your friend.


### Read the docs.

Functions that create or return copies of slices usually mention this in their documentation:

> "...returns a copy of..."
>
> "...returns a new byte slice..."

Whereas the documentation of in-place operations often talks about "slicing" or "subslices", which indicates that no allocation takes place and the returned data may still be accessed by other code.


## Links


**Happy coding!**

*/
