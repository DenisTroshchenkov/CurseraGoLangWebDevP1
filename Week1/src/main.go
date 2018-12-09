package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	out := new(bytes.Buffer)
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(out)
}

func dirTree(out *bytes.Buffer, dirName string, fullMod bool) error {
	err := dirTreeImpl(out, dirName, fullMod, "")
	if err != nil {
		return err
	}
	return nil
}
func dirTreeImpl(out *bytes.Buffer, dirName string, fullMod bool, indent string) error {
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		return err
	}
	var showFiles []os.FileInfo
	for _, file := range files {
		if file.IsDir() || fullMod {
			showFiles = append(showFiles, file)
		}
	}
	if len(showFiles) == 0 {
		return nil
	}
	for i := 0; i < len(showFiles)-1; i++ {
		printTreeBranch(out, showFiles[i], dirName, fullMod, indent, "├───", "│\t")
	}
	printTreeBranch(out, showFiles[len(showFiles)-1], dirName, fullMod, indent, "└───", "\t")
	return nil
}

func printTreeBranch(out *bytes.Buffer, file os.FileInfo, dirName string, fullMod bool, indent string, pref string, suff string) error {
	if file.IsDir() {
		out.WriteString(indent + pref + file.Name() + "\n")
		err := dirTreeImpl(out, filepath.Join(dirName, file.Name()), fullMod, indent+suff)
		if err != nil {
			return err
		}
	} else if fullMod {
		var fileSize string
		if file.Size() > 0 {
			fileSize = fmt.Sprintf("%db", file.Size())
		} else {
			fileSize = "empty"
		}
		out.WriteString(indent + pref + file.Name() + " (" + fileSize + ")\n")
	}
	return nil
}
