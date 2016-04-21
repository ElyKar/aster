package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	dir       string
	extension string
	recursive bool
	aggregate bool
)

type Stats struct {
	Code     int
	Comment  int
	Blank    int
	Total    int
	CodeS    float64
	CommentS float64
	BlankS   float64
}

func newStats(code, comment, blank int) *Stats {
	total := code + comment + blank
	return &Stats{
		code,
		comment,
		blank,
		total,
		float64(code) / float64(total) * 100,
		float64(comment) / float64(total) * 100,
		float64(blank) / float64(total) * 100,
	}
}

type Aggregator struct {
	Data map[string]*Stats
}

const agg = `
Total lines     {{.Total}}

Code lines      {{.Code}} / {{.CodeS | printf "%.2f"}}%
Comments        {{.Comment}} / {{.CommentS | printf "%.2f"}}%
Blank lines     {{.Blank}} / {{.BlankS | printf "%.2f"}}%
`

const split = `
{{range $key, $value := .Data}}{{$key}} : Total {{$value.Total}} ; Code {{$value.Code}}({{$value.CodeS | printf "%.2f"}}%) ; Comments {{$value.Comment}}({{$value.CommentS | printf "%.2f"}}%) ; Blank {{$value.Blank}}({{$value.BlankS | printf "%.2f"}}%)
{{else}}{{end}}`

var rootCmd = &cobra.Command{
	Use:   "aster [flags] file1 file2 ...",
	Short: "aster is a command-line tool to count the number of blank lines, comments or code lines in a set of files",
	Long:  `aster is a command-line tool for counting lines of code, commented lines and blank lines in a set of files`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(1)
		}

		a := &Aggregator{make(map[string]*Stats)}
		extensions := strings.Split(extension, ",")
		dirs := strings.Split(dir, ",")

		for _, filename := range args {
			if recursive {
				filepath.Walk(filename, func(filename string, info os.FileInfo, err error) error {
					if !info.IsDir() {
						walkFn(filename, extensions, a)
					} else if contains(dirs, filename) {
						return filepath.SkipDir
					}
					return nil
				})
			} else {
				walkFn(filename, extensions, a)
			}

		}

		if aggregate {
			codeAcc, commentAcc, blankAcc := 0, 0, 0
			for _, s := range a.Data {
				codeAcc += s.Code
				commentAcc += s.Comment
				blankAcc += s.Blank
			}
			tmpl := template.Must(template.New("stats").Parse(agg))
			_ = tmpl.Execute(os.Stdout, newStats(codeAcc, commentAcc, blankAcc))
		} else {
			tmpl := template.Must(template.New("stats").Parse(split))
			_ = tmpl.Execute(os.Stdout, a)
		}

	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("An error occurred ", err.Error())
		os.Exit(1)
	}

}

func init() {
	rootCmd.PersistentFlags().StringVarP(&dir, "exclude-dirs", "d", "", "exclude the list of comma-separated extensions, used with recursive search")
	rootCmd.PersistentFlags().StringVarP(&extension, "extension", "e", "", "search in all the files with this list of comma-separated extensions")
	rootCmd.PersistentFlags().BoolVarP(&aggregate, "aggregate", "a", false, "aggregate all the results, display info for each file by default")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "recursively search all the files in this sub-directory (do not follow symbolic links, do not recognize subtrees)")
}

func contains(s []string, elt string) bool {
	for _, e := range s {
		if strings.EqualFold(e, elt) {
			return true
		}
	}
	return false
}

func walkFn(filename string, extensions []string, a *Aggregator) {
	for _, ext := range extensions {
		if ext == "" || strings.HasSuffix(filename, "."+ext) {
			a.Data[filename] = newStats(visitFile(filename))
		}
	}
}

func visitFile(filename string) (code int, comment int, blank int) {
	content, _ := ioutil.ReadFile(filename)
	stringContent := string(content)
	comment, blank, code = 0, 0, 0
	lines := strings.Split(stringContent, "\n")
	state := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if state {
			if strings.HasSuffix(line, "*/") {
				state = false
			}
			comment += 1
		} else {
			if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
				comment += 1
			} else if strings.HasPrefix(line, "/*") {
				comment += 1
				state = true
			} else if line == "" {
				blank += 1
			} else {
				code += 1
			}
		}

	}
	blank -= 1 // The counted as blank
	return
}
