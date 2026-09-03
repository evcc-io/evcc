//go:build ignore

package main

// Extracts the serialized FileDescriptorProto bytes embedded in each *_pb2.py
// (the Python byte literal passed to AddSerializedFile) and writes a combined
// FileDescriptorSet to stdout (binary) or reconstructs .proto files.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// parsePyBytes converts a Python bytes literal body (the text between b'...' )
// into raw bytes. Handles \xHH, \n, \t, \r, \\, \', and literal chars.
func parsePyBytes(s string) []byte {
	var out []byte
	for i := 0; i < len(s); {
		c := s[i]
		if c != '\\' {
			out = append(out, c)
			i++
			continue
		}
		// escape
		i++
		if i >= len(s) {
			break
		}
		e := s[i]
		switch e {
		case 'x':
			// two hex digits
			h := s[i+1 : i+3]
			v, _ := strconv.ParseUint(h, 16, 8)
			out = append(out, byte(v))
			i += 3
		case 'n':
			out = append(out, '\n')
			i++
		case 't':
			out = append(out, '\t')
			i++
		case 'r':
			out = append(out, '\r')
			i++
		case '\\':
			out = append(out, '\\')
			i++
		case '\'':
			out = append(out, '\'')
			i++
		case '"':
			out = append(out, '"')
			i++
		case '0':
			out = append(out, 0)
			i++
		default:
			out = append(out, e)
			i++
		}
	}
	return out
}

var reCall = regexp.MustCompile(`(?s)AddSerializedFile\(\s*b'(.*?)'\s*\)`)

func main() {
	dir := os.Args[1]
	files, _ := filepath.Glob(filepath.Join(dir, "*_pb2.py"))
	var fds descriptorpb.FileDescriptorSet
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			panic(err)
		}
		m := reCall.FindSubmatch(data)
		if m == nil {
			fmt.Fprintf(os.Stderr, "no serialized file in %s\n", f)
			continue
		}
		raw := parsePyBytes(string(m[1]))
		var fdp descriptorpb.FileDescriptorProto
		if err := proto.Unmarshal(raw, &fdp); err != nil {
			fmt.Fprintf(os.Stderr, "unmarshal %s: %v\n", f, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "OK %s -> name=%s msgs=%d deps=%v\n", filepath.Base(f), fdp.GetName(), len(fdp.MessageType), fdp.GetDependency())
		fds.File = append(fds.File, &fdp)
	}
	// also need google well-known types timestamp & wrappers; include stubs if missing
	out, err := proto.Marshal(&fds)
	if err != nil {
		panic(err)
	}
	if len(os.Args) > 2 {
		_ = os.WriteFile(os.Args[2], out, 0644)
		fmt.Fprintf(os.Stderr, "wrote %d bytes to %s\n", len(out), os.Args[2])
	} else {
		os.Stdout.Write(out)
	}
	_ = strings.TrimSpace
}
