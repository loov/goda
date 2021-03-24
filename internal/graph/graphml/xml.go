package graphml

import (
	"bytes"
	"encoding/xml"
)

type File struct {
	XMLName           xml.Name `xml:"graphml"`
	XMLNS             string   `xml:"xmlns,attr"`
	XMLNSXSI          string   `xml:"xmlns:xsi,attr"`
	XMLNSY            string   `xml:"xmlns:y,attr"`
	XSISchemaLocation string   `xml:"xsi:schemalocation,attr"`

	Key    []Key    `xml:"key"`
	Graphs []*Graph `xml:"graph"`
}

func NewFile() *File {
	file := &File{}
	file.XMLNS = "http://graphml.graphdrawing.org/xmlns"
	file.XMLNSXSI = "http://www.w3.org/2001/XMLSchema-instance"
	file.XMLNSY = "http://www.yworks.com/xml/graphml"
	file.XSISchemaLocation = "http://graphml.graphdrawing.org/xmlns http://graphml.graphdrawing.org/xmlns/1.0/graphml.xsd"
	return file
}

type Graph struct {
	// XMLName xml.Name `xml:"graph"`
	ID          string      `xml:"id,attr"`
	EdgeDefault EdgeDefault `xml:"edgedefault,attr"`

	Node      []Node      `xml:"node"`
	Edge      []Edge      `xml:"edge"`
	Hyperedge []Hyperedge `xml:"hyperedge"`
	// TODO: parse info
}

type Key struct {
	ID  string `xml:"id,attr"`
	For string `xml:"for,attr"`

	AttrName string `xml:"attr.name,attr,omitempty"`
	AttrType string `xml:"attr.type,attr,omitempty"`

	YFilesType string `xml:"yfiles.type,attr,omitempty"`
}

type Node struct {
	// XMLName xml.Name `xml:"node"`
	ID    string   `xml:"id,attr"`
	Port  []Port   `xml:"port"`
	Graph []*Graph `xml:"graph"`
	Attrs Attrs    `xml:"data"`

	// TODO: parse info
}

type Port struct {
	// XMLName xml.Name `xml:"port"`
	Name string `xml:"name,attr"`
}

type Edge struct {
	// XMLName xml.Name `xml:"edge"`
	ID string `xml:"id,attr,omitempty"`

	Source   string `xml:"source,attr"`
	Target   string `xml:"target,attr"`
	Directed *bool  `xml:"directed,attr,omitempty"`

	SourcePort string `xml:"sourceport,attr,omitempty"`
	TargetPort string `xml:"targetport,attr,omitempty"`

	Attrs Attrs `xml:"data"`
}

type EdgeDefault string

const (
	Undirected = EdgeDefault("undirected")
	Directed   = EdgeDefault("directed")
)

type Attrs []Attr

func (attrs *Attrs) AddNonEmpty(key, value string) {
	if value == "" {
		return
	}
	*attrs = append(*attrs, Attr{Key: key, Value: escapeText(value)})
}

type Attr struct {
	// XMLName xml.Name `xml:"data"`
	Key   string `xml:"key,attr"`
	Value []byte `xml:",innerxml"`
}

type Hyperedge struct {
	// XMLName xml.Name `xml:"hyperedge"`

	ID       string     `xml:"id,attr,omitempty"`
	Endpoint []Endpoint `xml:"endpoint"`
}

type Endpoint struct {
	// XMLName xml.Name `xml:"endpoint"`
	Node string       `xml:"node,attr"`
	Port string       `xml:"port,attr,omitempty"`
	Type EndpointType `xml:"type,attr,omitempty"`
}

type EndpointType string

const (
	EndpointIn    = EndpointType("in")
	EndpointOut   = EndpointType("out")
	EndpointUndir = EndpointType("undir")
)

func escapeText(s string) []byte {
	if s == "" {
		return []byte{}
	}

	var buf bytes.Buffer
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		// this shouldn't ever happen
		panic(err)
	}
	return buf.Bytes()
}
