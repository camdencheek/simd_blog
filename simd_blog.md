---
id: simd_blog
aliases:
  - An Optimization Story
tags: []
---

# An Optimization Story

So, there's this function. It's called a lot. More importantly, all those calls are on the critical path of a key user interaction.
Let's talk about making it fast.

TODO: add link to code for this blog post

<aside>
Spoiler: it's a dot product.
</aside>

## Some background (or [skip to the juicy stuff](#the-target))

At Sourcegraph, we're working on a Code AI tool named [Cody](https://sourcegraph.com/cody). In order for Cody to answer questions well, we need to give it (him?) enough information to work with. One of the [ways we do this](https://about.sourcegraph.com/whitepaper/cody-context-architecture.pdf) is by leveraging [embeddings](https://platform.openai.com/docs/guides/embeddings).

An embedding is a vector representation of a chunk of text. They are constructed in such a way that semantically similar pieces of text are closer together. When Cody needs some more information, we run a similarity search over the embeddings to fetch a set of related chunks of code and feed those results to Cody to improve the quality of results.

The relevant part here is that similarity metric. For similarity search, a common metric is [cosine similarity](https://en.wikipedia.org/wiki/Cosine_similarity), which is just the cosine of the angle between two vectors. However, for normalized vectors (vectors with unit magnitude), cosine similarity yields a ranking equivalent to the [dot product](https://en.wikipedia.org/wiki/Dot_product). We do not index our embeddings ([yet](TODO)), so to run a search, we need to calculate the dot product for every embedding in our data set and keep the top results. And since we cannot start execution of the LLM until we get the necessary context, optimizing this step is crucial.

## The target

This is a simple implementation of a function that calculates the dot product of two vectors. My goal is to outline the journey I took to optimize this function, and to share some tools I picked up along the way.

```go
func DotNaive(a, b []float32) float32 {
	sum := float32(0)
	for i := 0; i < len(a) && i < len(b); i++ {
		sum += a[i] * b[i]
	}
	return sum
}
```

Unless otherwise stated, all benchmarks are run on an Intel Xeon Platinum 8481C 2.70GHz CPU. This is a `c3-highcpu-44` GCE VM.

## Loop unrolling

Modern CPUs do this thing called [_instruction pipelining_](https://en.wikipedia.org/wiki/Instruction_pipelining) where it can run multiple instructions simultaneously if it finds no data dependencies between them. A data dependency just means that the input of one instruction depends on the output of another.

In our simple implementation, we have data dependencies between our loop iterations. A couple, in fact. Both `i` and `sum` have a read/write pair each iteration, meaning an iteration cannot start executing until the previous is finished.

A common method of squeezing more out of our CPUs in situations like this is known as [_loop unrolling_](https://en.wikipedia.org/wiki/Loop_unrolling). The basic idea is to rewrite our loop so more of our relatively-high-latency multiply instructions can execute simultaneously.

```go
func DotUnroll4(a, b []float32) float32 {
	sum := float32(0)
	for i := 0; i < len(a); i += 4 {
		s0 := a[i] * b[i]
		s1 := a[i+1] * b[i+1]
		s2 := a[i+2] * b[i+2]
		s3 := a[i+3] * b[i+3]
		sum += s0 + s1 + s2 + s3
	}
	return sum
}
```

In our unrolled code, the dependencies between multiply instructions are removed, enabling the CPU to take more advantage of pipelining. This trims 27% from our naive implementation.

```
    │  naive.txt  │            unroll4.txt             │
    │   sec/op    │   sec/op     vs base               │
Dot   561.0m ± 0%   405.5m ± 0%  -27.71% (p=0.001 n=7)
```

Note that we can actually improve this slightly more by twiddling with the number of iterations we unroll. On my machine, 8 seemed to be optimal. However, the improvement is platform dependent and fairly minimal, so for the rest of the post, I'll stick with an unroll depth of 4 for readability.

## Bounds-checking elimination

In order to keep out-of-bounds slice accesses from being [a security vulnerability](https://en.wikipedia.org/wiki/Heartbleed), the go compiler inserts checks before each read. You can check it out in the [generated assembly](https://go.godbolt.org/z/qT3M7nPGf) (look for `runtime.panic`).

The compiled code makes it look like we wrote somthing like this:

```go
func DotUnroll4(a, b []float32) float32 {
	sum := float32(0)
	for i := 0; i < len(a); i += 4 {
        if i >= len(b) {
            panic("out of bounds")
        }
		s0 := a[i] * b[i]
        if i+1 >= len(a) || i+1 >= len(b) {
            panic("out of bounds")
        }
		s1 := a[i+1] * b[i+1]
        if i+2 >= len(a) || i+2 >= len(b) {
            panic("out of bounds")
        }
		s2 := a[i+2] * b[i+2]
        if i+3 >= len(a) || >= len(b) {
            panic("out of bounds")
        }
		s3 := a[i+3] * b[i+3]
		sum += s0 + s1 + s2 + s3
	}
	return sum
}
```

In a hot loop like this, even with modern branch prediction, the additional branches per iteration can add up to a pretty significant performance penalty. This is especially true in our case because the inserted jumps limit how much we can take advantage of pipelining.

To assess the impact of bounds checking, a trick I recently learned is to use the `-gcflags="-B"` option. It builds the binary without the bounds checks, allowing you to compare benchmarks with and without bounds checking. The comparison indicates that bounds checking accounts for roughly 2.5% of the remaining time.

```sh
$ benchstat <(go test -bench /unroll4 -count=5) <(go test -bench /unroll4 -count=5 -gcflags="-B")
               │   sec/op    │   sec/op     vs base              │
Dot/unroll4-44   412.0m ± 0%   402.0m ± 1%  -2.43% (p=0.002 n=6)
```

That's small enough that it probably wouldn't even be worth poking at. However, running this locally on an M1 mac yields a difference of over 30%, so I'm still going to go through the exercise of removing these checks.

If we can convince the compiler that these reads can never be out of bounds, it won't insert these runtime checks. This technique is known as "bounds-checking elimination", and the same pattern can be applied to many different memory-safe compiled languages.

In theory, we should be able to do all checks once, outside the loop, and the compiler would be able to determine that all the slice indexing is safe. However, I couldn't find the right combination of checks to convince the compiler that what I'm doing is safe. I landed on a combination of asserting the lengths are equal and moving all the bounds checking to the top of the loop. This was enough to hit nearly the speed of the bounds-check-free version.

```go
func DotBCE(a, b []float32) float32 {
	if len(a) != len(b) {
		panic("slices must have equal lengths")
	}

	sum := float32(0)
	for i := 0; i < len(a); i += 4 {
		aTmp := a[i : i+4 : i+4]
		bTmp := b[i : i+4 : i+4]
		s0 := aTmp[0] * bTmp[0]
		s1 := aTmp[1] * bTmp[1]
		s2 := aTmp[2] * bTmp[2]
		s3 := aTmp[3] * bTmp[3]
		sum += s0 + s1 + s2 + s3
	}
	return sum
}
```

Interestingly, the benchmark for this updated implementation shows a performance impact even greater than just using the `-B` flag! This one yields an 11% improvement compared to the 2.5% from `-B`. I actually don't have a good explanation for this. I'd love to hear if someone has an idea of why the difference is larger. 

<aside> 

Exercise for the reader: why is it significant that we slice like `a[i:i+4:i+4]` rather than just `a[i:i+4]`?

</aside>

This technique translates well to many memory-safe compiled languages like [Rust](https://nnethercote.github.io/perf-book/bounds-checks.html).

TODO: add discussion about `unsafe` package?

## Quantization

We've improved execution speed of our code pretty dramatically at this point, but there is another dimension of performance that is relevant here: memory usage.

(If you skipped the background section, this part might not make much sense.)

In our situation, we're searching over vectors with 1536 dimensions. With 4-byte elements, this comes out to 6KiB per vector, and we generate roughly a million vectors per GiB of code. That adds up.

When searching the vectors, they need to be held in RAM, which puts some serious memory pressure on our deployments. Moving the vectors out of memory would mean loading them from disk at search time, which is a no-go when performance is so important.

There are [plenty of ways](https://en.wikipedia.org/wiki/Dimensionality_reduction) to compress vectors, but we'll be talking about [_integer quantization_](https://huggingface.co/docs/optimum/concept_guides/quantization), which is relatively simple, but effective. The idea is to reduce the precision of the 4-byte `float32` vector elements by converting them to 1-byte `int8`s.

I won't get into exactly _how_ we decide to do the translation between `float32` and `int8`, since that's a pretty [deep topic](https://huggingface.co/docs/optimum/concept_guides/quantization), but suffice it to say our function now looks like the following:

```go
func DotInt8(a, b []int8) int32 {
	if len(a) != len(b) {
		panic("slices must have equal lengths")
	}

	sum := int32(0)
	for i := 0; i < len(a); i += 4 {
		aTmp := a[i : i+4 : i+4]
		bTmp := b[i : i+4 : i+4]
		s0 := int32(aTmp[0]) * int32(bTmp[0])
		s1 := int32(aTmp[1]) * int32(bTmp[1])
		s2 := int32(aTmp[2]) * int32(bTmp[2])
		s3 := int32(aTmp[3]) * int32(bTmp[3])
		sum += s0 + s1 + s2 + s3
	}
	return sum
}
```

This change yields a 4x reduction in memory usage at the cost of some recall precision (which we carefully measured, but is irrelevant to this blog post).

Re-running the benchmarks shows we suffer a perf hit from this change. Taking a look at the generated assembly (with `go tool compile -S`), there are a bunch of new instructions to convert `int8` to `int32`, so I expect that's the source of the slowdown. We'll make up for that in the next section.

## SIMD

I wanted an excuse to play with SIMD. And this problem seemed like the perfect nail for that hammer.

For those unfamiliar, SIMD stands for "Single Instruction Multiple Data". Basically, it lets you run an operation over a bunch of pieces of data with a single instruction. This is exactly what we want to do to calculate the dot product.

We have a problem though. Go does not expose SIMD intrinsics like [C](https://www.intel.com/content/www/us/en/docs/intrinsics-guide/index.html) or [Rust](https://doc.rust-lang.org/beta/core/simd/index.html). We have two options here: write it in C and use Cgo, or write it by hand for Go's assembler.
I try hard to avoid Cgo whenever possible for [many reasons that are not at all original](https://dave.cheney.net/2016/01/18/cgo-is-not-go), but one of those reasons is that Cgo imposes a performance penalty, and performance of this piece is paramount. Also, getting my hands dirty with some assembly sounds fun, so that's what I'm going to do.

I want this routine to be reasonably portable, so I'm going to restrict myself to only AVX2 instructions, which are supported on most `x86_64` server CPUs these days. We can use [runtime feature detection](TODO) to fall back to a slower option in pure Go.

The full code can be found [here](TODO). I'm not going to copy it all here, but I'll pick out some interesting tidbits.

The core loop of the implementation depends on three main instructions:

- `VPMOVSXBW`, which loads `int8`s into a vector `int16`s
- `VPMADDWD`, which multiplies two `int16` vectors element-wise, then adds together adjacent pairs to produce a vector of `int32`s
- `VPADDD`, which accumulates the resulting `int32` vector into our running sum

`VPMADDWD` is a real heavy lifter here. By combining the multiply and add steps into one, not only does it save instructions, it also helps us avoid overflow issues by simultaneously widening the result to an `int32`.

Let's see what this earned us.

```
cpu: Intel(R) Xeon(R) Platinum 8481C CPU @ 2.70GHz
    │   bce.txt    │             nosimd.txt              │              simd.txt              │
    │    sec/op    │    sec/op     vs base               │   sec/op     vs base               │
Dot   359.85m ± 1%   422.83m ± 0%  +17.50% (p=0.001 n=7)   75.16m ± 1%  -79.11% (p=0.001 n=7)
```

Woah! That's a 79% reduction from our previous best (before we switched to `int8`). SIMD for the win 🚀

Now, it wasn't all sunshine and rainbows. Hand-writing assembly in Go is weird. It uses a [custom assembler](https://go.dev/doc/asm), which means that its assembly language looks just-different-enough-to-be-confusing compared to the assembly snippets you usually find online. It has some weird quirks like [changing the order of instruction arguments](TODO) or [using different names for instructions](TODO). Some instructions don't even _have_ names in the go assembler and can only be used via their [binary encoding](TODO). Shameless plug: I found sourcegraph.com invaluable for finding examples of Go assembly to draw from.

That said, compared to Cgo, there are some nice benefits. Debugging still works well, the assembly can be stepped through, and registers can be inspected. There are no extra build steps (a C toolchain doesn't need to be set up). It's easy to set up a pure-Go fallback so cross-compilation still works. Common problems are caught by `go vet`.

## SIMD...but bigger

Previously, we limited ourselves to AVX2, but what if we _didn't_? The VNNI extension to AVX-512 added the [`VPDPBUSD`](https://www.felixcloutier.com/x86/vpdpbusd) instruction, which computes the dot product on `int8` vectors rather than `int16`s. This means we can process four times as many elements in a single instruction because we don't have to convert to `int16` first and our vector width doubles with AVX-512!

The only problem is that the instruction requires one vector to be signed bytes, and the other to be _unsigned_ bytes. Both of our vectors are signed. We can employ [a trick from Intel's developer guide](https://www.intel.com/content/www/us/en/docs/onednn/developer-guide-reference/2023-0/nuances-of-int8-computations.html#DOXID-DEV-GUIDE-INT8-COMPUTATIONS-1DG-I8-COMP-S12) to help us out. Basically, add 128 to one of our vectors to ensure it's in range of an unsigned integer, then keep track of how much overshoot we need to correct for at the end. The code can be found [here](TODO).

This implementation yielded another 21% improvement. Not bad!

```
    │  simd.txt   │              vnni.txt              │
    │   sec/op    │   sec/op     vs base               │
Dot   75.16m ± 1%   59.13m ± 1%  -21.33% (p=0.001 n=7)
```

## Bonus material

- If you haven't used [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat), you should. It's great. Super simple statistical comparison of benchmark results.
- I got [nerd sniped](https://twitter.com/sluongng/status/1654066471230636033) into implementing a [version for ARM](TODO), which made for an interesting comparison.
- To avoid distributing multiple binaries, we take advantage of [runtime feature detection](TODO) to seamlessly switch between versions on start up.
- If you haven't come across it, the [Agner Fog Instruction Tables](https://www.agner.org/optimize/) make for some great reference material for low-level optimizations.
- Don't miss the [compiler explorer](https://go.godbolt.org/z/qT3M7nPGf), which is an extremely useful tool for digging into compiler codegen.
