package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	fc "github.com/runZeroInc/gosnmp/pkg/fuzzcache"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s [pack|unpack] <go-fuzz-cache> <testdata-fuzzcache>\n", os.Args[0])
		os.Exit(1)
	}
	mode := os.Args[1]
	goCacheDir := os.Args[2]
	testCacheDir := os.Args[3]

	switch mode {
	case "pack":
		fmt.Printf("Packing fuzz inputs from Go fuzz cache %s into testdata fuzz cache %s\n", goCacheDir, testCacheDir)
		err := PackFuzzInputs(goCacheDir, testCacheDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error packing fuzz inputs: %v\n", err)
			os.Exit(1)
		}
	case "unpack":
		fmt.Printf("Unpacking fuzz inputs from testdata fuzz cache %s into Go fuzz cache %s\n", testCacheDir, goCacheDir)
		err := UnpackFuzzInputs(goCacheDir, testCacheDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error unpacking fuzz inputs: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s. Use 'pack' or 'unpack'.\n", mode)
		os.Exit(1)
	}
}

func PackFuzzInputs(goCacheDir, testCacheDir string) error {
	ents, err := os.ReadDir(goCacheDir)
	if err != nil {
		return err
	}
	for _, ent := range ents {
		st, err := ent.Info()
		if err != nil {
			return err
		}
		if !st.IsDir() {
			continue
		}
		res, err := fc.LoadGoFuzzCache(goCacheDir, ent.Name())
		if err != nil {
			return err
		}
		fmt.Printf("Loaded %d inputs from Go fuzz cache for %s\n", len(res), ent.Name())

		cur, err := fc.LoadFuzzInputsFromCache(testCacheDir, ent.Name())
		if err != nil {
			return err
		}
		fmt.Printf("Loaded %d existing inputs from testdata for %s\n", len(cur), ent.Name())

		res = append(res, cur...)

		fmt.Printf("Saving %d inputs to testdata fuzz cache for %s\n", len(res), ent.Name())
		err = fc.SaveFuzzInputsToCache(testCacheDir, ent.Name(), res)
		if err != nil {
			return err
		}
	}
	return nil
}

func UnpackFuzzInputs(goCacheDir, testCacheDir string) error {
	ents, err := os.ReadDir(testCacheDir)
	if err != nil {
		return err
	}
	for _, ent := range ents {
		st, err := ent.Info()
		if err != nil {
			return err
		}
		if st.IsDir() {
			continue
		}
		if !strings.HasSuffix(ent.Name(), ".fuzz") {
			continue
		}
		fname := strings.TrimSuffix(ent.Name(), ".fuzz")
		cur, err := fc.LoadFuzzInputsFromCache(testCacheDir, fname)
		if err != nil {
			return err
		}
		fmt.Printf("Loaded %d existing inputs from testdata for %s\n", len(cur), ent.Name())

		fmt.Printf("Saving %d inputs to Go fuzz cache for %s\n", len(cur), fname)
		err = os.MkdirAll(filepath.Join(goCacheDir, fname), 0755)
		if err != nil {
			return err
		}
		err = fc.SaveGoFuzzCacheFile(filepath.Join(goCacheDir, fname), cur)
		if err != nil {
			return err
		}
	}

	return nil
}
