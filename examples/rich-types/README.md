# rich-types: strings, byte arrays and scalars across the boundary

This example proves the second boundary mode. `permctl` keeps everything scalar
and gets a ~27 KB runtime-free archive. Here `String` and `ByteArray` values
cross the boundary, so LeanPort links the Lean runtime into a single static
archive (~29 MB) and the generated Go package stays a normal Go package.

```sh
leanport check
leanport build
go test ./...
go run ./cmd/richdemo
```

Expected output:

```
Hello, LeanPort!
sum([1 2 3 250]) = 256
reverse("leanport") = "tropnael"
```

## What is generated

| Lean export | Go signature |
|---|---|
| `leanport_rich_greet : String → String` | `Greet(string) string` |
| `leanport_rich_sum : ByteArray → UInt64` | `Sum([]byte) uint64` |
| `leanport_rich_reverse : ByteArray → ByteArray` | `Reverse([]byte) []byte` |

The bridge is precompiled into `internal/verified/rich/native/linux_amd64/libRich.a`.
It initializes Lean once (`lean_initialize_runtime_module`, module initializers,
`lean_io_mark_end_initialization`), converts Go strings/bytes into owned Lean
objects, copies results back, and releases Lean references. Consumers do not
include `lean.h` and need no Lean toolchain.

Because the archive is ~29 MB, it is **not committed**; run `leanport build`
after cloning. `permctl`'s runtime-free archive is committed so the capstone
example works with `go test ./...` alone.
