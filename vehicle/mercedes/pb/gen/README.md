# Mercedes protobuf regeneration tooling

One-off tooling used to regenerate the `vehicle/mercedes/pb` bindings from the
[ReneNulschDE/mbapi2020](https://github.com/ReneNulschDE/mbapi2020) descriptors.
These files carry a `//go:build ignore` tag so they are excluded from the normal
build.

## Steps

```sh
# 1. Clone the reference integration.
git clone --depth 1 https://github.com/ReneNulschDE/mbapi2020 /tmp/mbapi2020

# 2. Extract the embedded FileDescriptorProtos from the compiled *_pb2.py files
#    into a single FileDescriptorSet.
go run extract.go /tmp/mbapi2020/custom_components/mbapi2020/proto /tmp/mbfds.bin

# 3. Drive protoc-gen-go from the descriptor set, applying evcc's go_package
#    layout and stripping the gogo.proto custom options.
go run gen.go /tmp/mbfds.bin "$(go env GOPATH)/bin/protoc-gen-go" /tmp/mbgen

# 4. Copy the result into place: protos.pb.go -> pb/protos/, everything else -> pb/.
```

`extract.go` parses the Python byte literals passed to
`AddSerializedFile(b'...')` in each `*_pb2.py` and unmarshals them into
`descriptorpb.FileDescriptorProto`. `gen.go` adds the well-known-type
descriptors, remaps `go_package`, removes the `gogo.proto` dependency and its
extension options, then pipes a `CodeGeneratorRequest` to `protoc-gen-go`.
