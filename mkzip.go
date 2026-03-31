//go:build ignore

// Mkzip writes a zoneinfo.zip with the content of the current directory
// and its subdirectories, with no compression, suitable for embedding.
//
// Usage:
//
//	go run mkzip.go zoneinfo.zip
package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"hash/crc32"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: go run mkzip.go zoneinfo.zip\n")
	os.Exit(2)
}

func main() {
	log.SetPrefix("mkzip: ")
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 || !strings.HasSuffix(args[0], ".zip") {
		usage()
	}

	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	seen := make(map[string]bool)
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		if strings.HasSuffix(path, ".zip") {
			log.Fatalf("unexpected file during walk: %s", path)
		}
		name := filepath.ToSlash(path)
		w, err := zw.CreateRaw(&zip.FileHeader{
			Name:               name,
			Method:             zip.Store,
			CompressedSize64:   uint64(len(data)),
			UncompressedSize64: uint64(len(data)),
			CRC32:              crc32.ChecksumIEEE(data),
		})
		if err != nil {
			log.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			log.Fatal(err)
		}
		seen[name] = true
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}
	if len(seen) == 0 {
		log.Fatalf("did not find any files to add")
	}
	if !seen["US/Eastern"] {
		log.Fatalf("did not find US/Eastern to add")
	}
	if err := os.WriteFile(args[0], zb.Bytes(), 0666); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("mkzip: wrote %s (%d files, %d bytes)\n", args[0], len(seen), zb.Len())
}
