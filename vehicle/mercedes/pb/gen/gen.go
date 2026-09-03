//go:build ignore

package main

// Reads the extracted FileDescriptorSet, adds well-known-type descriptors,
// applies go_package (M) mappings matching evcc's layout, and pipes a
// CodeGeneratorRequest to protoc-gen-go.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	// import well-known types so their descriptors are registered
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/emptypb"
	_ "google.golang.org/protobuf/types/known/structpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	_ "google.golang.org/protobuf/types/known/wrapperspb"
)

const evccPkg = "github.com/evcc-io/evcc/vehicle/mercedes/pb"

var wktSet = map[string]bool{
	"google/protobuf/timestamp.proto":  true,
	"google/protobuf/wrappers.proto":   true,
	"google/protobuf/struct.proto":     true,
	"google/protobuf/descriptor.proto": true,
	"google/protobuf/duration.proto":   true,
	"google/protobuf/empty.proto":      true,
	"google/protobuf/any.proto":        true,
}

// go_package mapping: all files -> pb package "protos" at evccPkg, except
// protos.proto which lives in the protos/ subdir (also package protos).
func goPackageFor(name string) string {
	switch name {
	case "protos.proto":
		return evccPkg + "/protos;protos"
	case "gogo.proto":
		return "github.com/gogo/protobuf/gogoproto;gogoproto"
	}
	return evccPkg + ";protos"
}

func main() {
	fdsBytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(fdsBytes, &fds); err != nil {
		panic(err)
	}

	byName := map[string]*descriptorpb.FileDescriptorProto{}
	for _, f := range fds.File {
		byName[f.GetName()] = f
	}

	for w := range wktSet {
		if _, ok := byName[w]; ok {
			continue
		}
		fd, err := protoregistry.GlobalFiles.FindFileByPath(w)
		if err != nil {
			continue
		}
		byName[w] = protodesc.ToFileDescriptorProto(fd)
	}

	// Drop the gogo.proto dependency and any gogo extension options. The gogo
	// extensions are custom field/message options that do not affect the wire
	// format; stripping them lets protoc-gen-go generate without importing
	// github.com/gogo/protobuf (matching evcc's existing pb layout).
	delete(byName, "gogo.proto")
	for _, f := range byName {
		sanitizeFile(f)
		// Never rewrite the go_package of well-known types – they must keep
		// their canonical import path (wrapperspb, timestamppb, …).
		if wktSet[f.GetName()] {
			continue
		}
		if f.Options == nil {
			f.Options = &descriptorpb.FileOptions{}
		}
		f.Options.GoPackage = proto.String(goPackageFor(f.GetName()))
	}

	genFiles := []string{}
	for n := range byName {
		if n == "gogo.proto" || wktSet[n] {
			continue
		}
		genFiles = append(genFiles, n)
	}
	sort.Strings(genFiles)

	sorted := topoSort(byName)

	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate:  genFiles,
		ProtoFile:       sorted,
		CompilerVersion: &pluginpb.Version{Major: proto.Int32(5), Minor: proto.Int32(29), Patch: proto.Int32(0)},
		Parameter:       proto.String("paths=source_relative"),
	}
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		panic(err)
	}

	cmd := exec.Command(os.Args[2])
	cmd.Stdin = bytes.NewReader(reqBytes)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "protoc-gen-go stderr:\n%s\n", ee.Stderr)
		}
		panic(err)
	}
	var resp pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(out, &resp); err != nil {
		panic(err)
	}
	if resp.GetError() != "" {
		fmt.Fprintf(os.Stderr, "codegen error: %s\n", resp.GetError())
		os.Exit(1)
	}
	outDir := os.Args[3]
	for _, f := range resp.File {
		p := filepath.Join(outDir, f.GetName())
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(p, []byte(f.GetContent()), 0644); err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", p, len(f.GetContent()))
	}
}

// sanitizeFile removes the gogo.proto dependency and clears extension options
// (which carry gogo custom options) throughout the descriptor tree.
func sanitizeFile(f *descriptorpb.FileDescriptorProto) {
	// Remove gogo.proto from the dependency list and fix public/weak indexes.
	deps := f.GetDependency()
	kept := deps[:0]
	for _, d := range deps {
		if d == "gogo.proto" {
			continue
		}
		kept = append(kept, d)
	}
	f.Dependency = kept
	f.PublicDependency = nil
	f.WeakDependency = nil

	if f.Options != nil {
		f.Options.ProtoReflect().SetUnknown(nil)
	}
	for _, m := range f.MessageType {
		sanitizeMessage(m)
	}
	for _, e := range f.EnumType {
		if e.Options != nil {
			e.Options.ProtoReflect().SetUnknown(nil)
		}
		for _, v := range e.Value {
			if v.Options != nil {
				v.Options.ProtoReflect().SetUnknown(nil)
			}
		}
	}
	for _, s := range f.Service {
		if s.Options != nil {
			s.Options.ProtoReflect().SetUnknown(nil)
		}
		for _, mth := range s.Method {
			if mth.Options != nil {
				mth.Options.ProtoReflect().SetUnknown(nil)
			}
		}
	}
}

func sanitizeMessage(m *descriptorpb.DescriptorProto) {
	if m.Options != nil {
		m.Options.ProtoReflect().SetUnknown(nil)
	}
	for _, fld := range m.Field {
		if fld.Options != nil {
			fld.Options.ProtoReflect().SetUnknown(nil)
		}
	}
	for _, e := range m.EnumType {
		if e.Options != nil {
			e.Options.ProtoReflect().SetUnknown(nil)
		}
		for _, v := range e.Value {
			if v.Options != nil {
				v.Options.ProtoReflect().SetUnknown(nil)
			}
		}
	}
	for _, nested := range m.NestedType {
		sanitizeMessage(nested)
	}
}

func topoSort(byName map[string]*descriptorpb.FileDescriptorProto) []*descriptorpb.FileDescriptorProto {
	var out []*descriptorpb.FileDescriptorProto
	visited := map[string]bool{}
	var visit func(n string)
	visit = func(n string) {
		if visited[n] {
			return
		}
		f, ok := byName[n]
		if !ok {
			return
		}
		visited[n] = true
		for _, d := range f.GetDependency() {
			visit(d)
		}
		out = append(out, f)
	}
	names := []string{}
	for n := range byName {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		visit(n)
	}
	return out
}
