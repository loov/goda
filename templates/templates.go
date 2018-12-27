package templates

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/template"

	"github.com/loov/goda/memory"
	"golang.org/x/tools/go/packages"
)

func Parse(t string) (*template.Template, error) {
	return template.New("").Funcs(template.FuncMap{
		"LineCount":  LineCount,
		"SourceSize": SourceSize,
		"AllFiles":   AllFiles,
	}).Parse(t)
}

func LineCount(vs ...interface{}) int64 {
	var count int64

	for _, v := range vs {
		var files []string
		switch v := v.(type) {
		case []string: // assume we want the count of a list of files
			files = v
		case *packages.Package: // assume we want the count of all files in package directories
			files = allFiles(v)
		}

		for _, filename := range files {
			r, err := os.Open(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v open failed: %v", filename, err)
				continue
			}
			count += countLines(r)

			if err := r.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "%v close failed: %v", filename, err)
				continue
			}
		}
	}

	return count
}

func SourceSize(vs ...interface{}) memory.Bytes {
	var size int64

	for _, v := range vs {
		var files []string
		switch v := v.(type) {
		case []string: // assume we want the size of a list of files
			files = v
		case *packages.Package: // assume we want the size of all files in package directories
			files = allFiles(v)
		}

		for _, filename := range files {
			stat, err := os.Stat(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v stat failed: %v", filename, err)
				continue
			}
			size += stat.Size()
		}
	}

	return memory.Bytes(size)
}

func AllFiles(vs ...interface{}) []string {
	var files []string
	for _, v := range vs {
		switch v := v.(type) {
		case []string: // assume we want the size of a list of files
			files = append(files, v...)
		case *packages.Package: // assume we want the size of all files in package directories
			files = append(files, allFiles(v)...)
		}
	}
	return files
}

func allFiles(p *packages.Package) []string {
	files := map[string]bool{}
	for _, filename := range p.GoFiles {
		files[filename] = true
	}
	for _, filename := range p.OtherFiles {
		files[filename] = true
	}

	var list []string
	for file := range files {
		list = append(list, file)
	}
	sort.Strings(list)

	return list
}

func countLines(r io.Reader) int64 {
	var count int64
	var buffer [1 << 20]byte
	for {
		n, err := r.Read(buffer[:])

		for _, r := range buffer[:n] {
			if r == 0 { // probably a binary file
				return 0
			}
			if r == '\n' {
				count++
			}
		}

		if err != nil || n == 0 {
			return count
		}
	}
}
