package gosnmp

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func LoadFuzzInputsFromCache(cacheDir, funcName string) ([][]byte, error) {
	srcFile := filepath.Join(cacheDir, funcName+".fuzz")
	fd, err := os.Open(srcFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open cache file %s: %v", srcFile, err)
	}
	defer fd.Close()
	var inputs [][]byte
	scanner := bufio.NewReader(fd)
	lastLine := []byte{}
	for {
		line, isPrefix, err := scanner.ReadLine()
		if isPrefix {
			lastLine = append(lastLine, line...)
			continue
		}

		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read cache file %s: %v", srcFile, err)
		}

		line = append(lastLine, line...)
		lastLine = lastLine[:0]

		if len(line) == 0 {
			if err == io.EOF {
				break
			}
			continue
		}

		data, err := strconv.Unquote(string(line))
		if err != nil {
			log.Printf("failed to unquote data %s: %v", line, err)
			continue
		}
		inputs = append(inputs, []byte(data))

		if err == io.EOF {
			break
		}
	}
	return inputs, nil
}

func SaveFuzzInputsToCache(cacheDir, funcName string, inputs [][]byte) error {
	dstFile := filepath.Join(cacheDir, funcName+".fuzz")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return fmt.Errorf("failed to create cache dir %s: %v", cacheDir, err)
	}
	fd, err := os.Create(dstFile)
	if err != nil {
		return fmt.Errorf("failed to create cache file %s: %v", dstFile, err)
	}
	defer fd.Close()

	// Deduplicate and sort inputs
	umap := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		umap[string(input)] = struct{}{}
	}
	uvals := make([]string, 0, len(umap))
	for s := range umap {
		uvals = append(uvals, s)
	}
	sort.Strings(uvals)

	for _, input := range uvals {
		_, err := fd.WriteString(fmt.Sprintf("%q\n", string(input)))
		if err != nil {
			return fmt.Errorf("failed to write to cache file %s: %v", dstFile, err)
		}
	}
	return nil
}

// LoadGoFuzzCache loads []byte inputs from the specified Go cache and function name.
func LoadGoFuzzCache(goFuzzCacheDir, funcName string) ([][]byte, error) {
	var inputs [][]byte
	baseDir := filepath.Join(goFuzzCacheDir, funcName)
	err := fs.WalkDir(os.DirFS(baseDir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		loadedInput, err := LoadGoFuzzCacheFile(filepath.Join(baseDir, path))
		if err != nil {
			return fmt.Errorf("failed to load fuzz cache file %s: %v", path, err)
		}
		inputs = append(inputs, loadedInput)
		return nil
	})
	return inputs, err
}

// LoadGoFuzzCacheFile loads the first []byte input from a Go fuzz cache file.
func LoadGoFuzzCacheFile(fname string) ([]byte, error) {
	var input []byte
	f, err := os.Open(fname)
	if err != nil {
		return input, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "[]byte(") && strings.HasSuffix(line, ")") {
			dataStr := line[len("[]byte(") : len(line)-1]
			data, err := strconv.Unquote(dataStr)
			if err != nil {
				log.Printf("failed to unquote data %s in line %q: %v", dataStr, line, err)
				continue
			}
			return []byte(data), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return input, err
	}
	return input, nil
}

// encVersion1 will be the first line of a file with version 1 encoding.
var encVersion1 = "go test fuzz v1"

// SaveGoFuzzCacheFile write a go fuzz v1 files for single []byte inputs.
func SaveGoFuzzCacheFile(baseDir string, data [][]byte) error {
	for _, tc := range data {

		// https://cs.opensource.google/go/go/+/master:src/internal/fuzz/encoding.go;l=19?q=encVersion1&ss=go%2Fgo
		data := fmt.Appendf(nil, "%s\n[]byte(%q)\n", encVersion1, tc)

		// https://cs.opensource.google/go/go/+/master:src/internal/fuzz/fuzz.go;l=1049
		dsum := sha256.Sum256(data)
		name := fmt.Sprintf("%x", dsum)[:16]
		f, err := os.Create(filepath.Join(baseDir, name))
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}
