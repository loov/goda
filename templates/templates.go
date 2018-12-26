package templates

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"text/template"

	"golang.org/x/tools/go/packages"
)

func Parse(t string) (*template.Template, error) {
	return template.New("").Funcs(template.FuncMap{
		"LineCount": LineCount,
		"Size":      Size,
		"AllFiles":  AllFiles,
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

func Size(vs ...interface{}) string {
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

	return bytes(size)
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
	dirs := map[string]bool{}
	for _, filename := range p.GoFiles {
		dirs[filepath.Dir(filename)] = true
	}
	for _, filename := range p.OtherFiles {
		dirs[filepath.Dir(filename)] = true
	}

	var files []string
	for dir := range dirs {
		file, err := os.Open(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v open failed: %v", dir, err)
			continue
		}

		stats, err := file.Readdir(-1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v readdirnames failed: %v", dir, err)
			continue
		}

		for _, stat := range stats {
			if stat.IsDir() {
				continue
			}

			files = append(files, filepath.Join(dir, stat.Name()))
		}
	}
	sort.Strings(files)

	return files
}

func bytes(s int64) string {
	size := float64(s)

	switch {
	case size >= (1<<60)*2/3:
		return fmt.Sprintf("%.1fEB", size/(1<<60))
	case size >= (1<<50)*2/3:
		return fmt.Sprintf("%.1fPB", size/(1<<50))
	case size >= (1<<40)*2/3:
		return fmt.Sprintf("%.1fTB", size/(1<<40))
	case size >= (1<<30)*2/3:
		return fmt.Sprintf("%.1fGB", size/(1<<30))
	case size >= (1<<20)*2/3:
		return fmt.Sprintf("%.1fMB", size/(1<<20))
	case size >= (1<<10)*2/3:
		return fmt.Sprintf("%.1fKB", size/(1<<10))
	}

	return strconv.Itoa(int(s)) + " B"
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
