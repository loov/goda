package stat

import (
	"errors"
	"os"

	"github.com/loov/goda/internal/memory"
)

// Source contains basic analysis of arbitrary source code.
type Source struct {
	// Files count in this stat.
	Files int
	// Binary file count.
	Binary int
	// Size in bytes of all files.
	Size memory.Bytes
	// Count of non-empty lines.
	Lines int
	// Count of empty lines.
	Blank int
}

func (c *Source) Add(s Source) {
	c.Files += s.Files
	c.Binary += s.Binary
	c.Size += s.Size

	c.Blank += s.Blank
	c.Lines += s.Lines
}

func (c *Source) Sub(s Source) {
	c.Files -= s.Files
	c.Binary -= s.Binary
	c.Size -= s.Size

	c.Blank -= s.Blank
	c.Lines -= s.Lines
}

func SourceFromBytes(data []byte) Source {
	count := Source{Files: 1}
	if len(data) == 0 {
		return count
	}
	count.Size += memory.Bytes(len(data))

	emptyline := true
	for _, c := range data {
		switch c {
		case 0x0:

			count.Blank = 0
			count.Lines = 0
			count.Files = 0
			count.Binary = 1

			return count
		case '\n':
			if emptyline {
				count.Blank++
			} else {
				count.Lines++
			}
			emptyline = true
		case '\r', ' ', '\t': // ignore
		default:
			emptyline = false
		}
	}
	if !emptyline {
		count.Lines++
	}

	return count
}

var ErrEmptyFile = errors.New("empty file")

func SourceFromPath(path string) (Source, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Source{}, err
	}
	if len(data) == 0 {
		return Source{Files: 1}, ErrEmptyFile
	}
	return SourceFromBytes(data), nil
}
