// Copyright 2021 Saferwall. All rights reserved.
// Use of this source code is governed by Apache v2 license
// license that can be found in the LICENSE file.

package utils

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
)

// WriteStrSliceToFile writes a slice of string line by line to a file.
func WriteStrSliceToFile(filename string, data []string) (int, error) {
	// Open a new file for writing only
	file, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Create a new writer.
	w := bufio.NewWriter(file)
	nn := 0
	for _, s := range data {
		n, _ := w.WriteString(s + "\n")
		nn += n
	}

	w.Flush()
	return nn, nil
}

// Read a whole file into the memory and store it as array of lines
func ReadLines(path string) (lines []string, err error) {

	var (
		part   []byte
		prefix bool
	)

	// Start by getting a file descriptor over the file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	buffer := bytes.NewBuffer(make([]byte, 0))
	for {
		if part, prefix, err = reader.ReadLine(); err != nil {
			break
		}
		buffer.Write(part)
		if !prefix {
			lines = append(lines, buffer.String())
			buffer.Reset()
		}
	}
	if err == io.EOF {
		err = nil
	}
	return
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// StringInSlice returns whether or not a string exists in a slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
func regexp2FindAllString(re *regexp2.Regexp, s string) []string {
	var matches []string
	m, _ := re.FindStringMatch(s)
	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}
	return matches
}

func RegSubMatchToMapString(regEx, s string) (paramsMap map[string]string) {

	r := regexp.MustCompile(regEx)
	match := r.FindStringSubmatch(s)

	paramsMap = make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return
}

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// Group multi-whitespaces to one whitespace.
func StandardizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Strip all whitespaces.
func SpaceFieldsJoin(s string) string {
	return strings.Join(strings.Fields(s), "")
}

// Remove C language comments.
// Removes both single line and multi-line comments.
func StripComments(s string) string {

	// Remove first the single line ones.
	regSingleLine := regexp.MustCompile(`//.*`)
	s = regSingleLine.ReplaceAllString(s, "")

	// Then the multi-lines ones.
	regMultiLine := regexp.MustCompile(`/\*([^*]|[\r\n]|(\*+([^*/]|[\r\n])))*\*+/`)
	s = regMultiLine.ReplaceAllString(s, "")
	return s
}

func FindClosingBracket(text []byte, openPos int) int {
	closePos := openPos
	counter := 1
	for counter > 0 {
		closePos++
		c := text[closePos]
		if c == '{' {
			counter++
		} else if c == '}' {
			counter--
		}
	}
	return closePos
}

func FindClosingSemicolon(text []byte, pos int) int {
	for text[pos] != ';' {
		pos++
	}
	return pos
}

// ReadAll reads the entire file into memory.
func ReadAll(filePath string) ([]byte, error) {
	// Start by getting a file descriptor over the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Get the file size to know how much we need to allocate
	fileinfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	filesize := fileinfo.Size()
	buffer := make([]byte, filesize)

	// Read the whole binary
	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

// WalkAllFilesInDir returns list of files in directory.
func WalkAllFilesInDir(dir string) ([]string, error) {

	fileList := []string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}

		// check if it is a regular file (not dir)
		if info.Mode().IsRegular() {
			fileList = append(fileList, path)
		}
		return nil
	})

	return fileList, err
}

// WriteBytesFile write Bytes to a File.
func WriteBytesFile(filename string, r io.Reader) (int, error) {

	// Open a new file for writing only
	file, err := os.OpenFile(
		filename,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Read for the reader bytes to file
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return 0, err
	}

	// Write bytes to disk
	bytesWritten, err := file.Write(b)
	if err != nil {
		return 0, err
	}

	return bytesWritten, nil
}
