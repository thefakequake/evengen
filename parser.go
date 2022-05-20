package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"
)

const (
	h1 = "# "
	h2 = "## "
	h3 = "### "
	h4 = "#### "
	h6 = "###### "
)

// stolen from https://github.com/writeas/go-strip-markdown/blob/master/strip.go
var removeMarkdownURLRegexp = regexp.MustCompile(`\[(.*?)\][\[\(].*?[\]\)]`)

func ParseMarkdown(cfg *Config) {
	files, err := ioutil.ReadDir(cfg.OutDir + "/md")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		f, err := os.Open(cfg.OutDir + "/md/" + name)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		dat, err := io.ReadAll(f)
		if err != nil {
			log.Fatal(err)
		}

		parsed := ParseFile(string(dat))
		if len(parsed) < 2 {
			fmt.Printf("parsed %s but found no structs\n", name)
			continue
		}

		var imports string
		if strings.Contains(parsed, "time.Time") {
			imports = "\n\nimport (\n	\"time\"\n)"
		}

		parsed = fmt.Sprintf("package %s%s\n%s", cfg.Package, imports, parsed)
		goFileName := strings.TrimSuffix(strings.ReplaceAll(strings.ToLower(name), "_", ""), "md") + "go"
		if err := os.WriteFile(cfg.OutDir+"/go/"+goFileName, []byte(parsed), 0644); err != nil {
			log.Fatal(err)
		}

		fmt.Printf("parsed %s -> %s\n", name, goFileName)
	}
}

func ParseFile(text string) string {
	uncleansedLines := strings.Split(text, "\n")
	url := uncleansedLines[0]

	var lines []string
	for _, l := range uncleansedLines[1:] {
		trim := strings.TrimSpace(l)
		if len(trim) == 0 {
			continue
		}
		lines = append(lines, trim)
	}
	// inserts empty line at end of file in order to allow final table to be parsed
	lines = append(lines, "")

	var currentLine int
	var parsingTable bool
	var tables []Table
	var currentTable Table
	var lastHeader string

	for currentLine < len(lines) {
		cur := lines[currentLine]
		currentLine++

		if parsingTable {
			if strings.HasPrefix(cur, "|") {
				row := SplitTableRow(cur)
				if len(row) < 3 {
					continue
				} else if len(row) > 3 {
					row = row[:3]
				}

				var optional bool
				if strings.HasSuffix(row[0], "?") {
					optional = true
				}

				for i := 0; i < 3; i++ {
					row[i] = strings.Trim(row[i], "?*\\ ")
				}

				if ContainsWord(strings.Split(row[2], " "), "deprecated") {
					continue
				}

				currentTable.Fields = append(currentTable.Fields, Field{RemoveHyperlinks(row[0]), RemoveHyperlinks(row[1]), RemoveHyperlinks(row[2]), optional})
			} else if strings.HasPrefix(cur, ">") || len(currentTable.Fields) == 0 {
				continue
			} else {
				if len(currentTable.Fields) > 0 {
					tables = append(tables, currentTable)
				}
				currentTable = Table{}
				parsingTable = false
			}
		}

		title := strings.TrimLeft(cur, "# ")

		if (strings.Contains(title, "Example")) {
			continue
		}
		if strings.HasPrefix(cur, h6) {
			for _, p := range []string{"Structure", "Object", "Metadata", "Info"} {
				if strings.Contains(title, p) {
					parsingTable = true
					currentTable.URL = url + "#" + strcase.ToKebab(lastHeader+title)
					
					t := title
					if !ContainsWord([]string{"Metadata", "Info"}, p) {
						t = RemoveSuffix(t, p)
					}
					currentTable.Title = t
				}
			}
		}
		for _, h := range []string{h1, h2, h3, h4} {
			if strings.HasPrefix(cur, h) {
				lastHeader = title
				break
			}
		}
	}

	var file string
	for _, t := range tables {
		file += "\n" + t.ToString() + "\n"
	}

	return file
}

type Table struct {
	Title  string
	URL    string
	Fields []Field
}

type Field struct {
	Name        string
	Type        string
	Description string
	Optional    bool
}

func (t Table) ToString() string {
	tableStr := fmt.Sprintf("// %s\ntype %s struct {", t.URL, ConvertName(t.Title))
	for _, f := range t.Fields[2:] {
		var empty string
		if f.Optional {
			empty = ",omitempty"
		}
		tableStr += fmt.Sprintf("\n	// %s\n	%s %s `json:\"%s%s\"`\n", ConvertDesc(f.Description), ConvertName(f.Name), ParseType(f.Type), f.Name, empty)
	}
	tableStr += "}"

	return tableStr
}

func ParseType(t string) string {
	var split []string

	for _, w := range strings.Split(strings.TrimLeft(t, "?"), " ") {
		if strings.HasPrefix(w, "(") {
			break
		}
		split = append(split, w)
	}

	if len(split) == 1 {
		switch split[0] {
		case "string":
			return "string"
		case "snowflake":
			return "string"
		case "integer":
			return "int"
		case "int":
			return "int"
		case "float":
			return "float64"
		case "boolean":
			return "bool"
		case "null":
			return "bool"
		case "mixed":
			return "interface{}"
		}
	} else if len(split) >= 2 {
		start := strings.Join(split[:2], " ")
		if start == "array of" || start == "list of" {
			return "[]" + ParseType(strings.TrimSuffix(strings.Join(split[2:], " "), "s"))
		} else if start == "dictionary with" {
			return "map[string]string"
		} else if start == "one of" {
			return ConvertName(strings.Join(split[2:], " "))
		} else if start == "image data" {
			return "string"
		} else if len(split) == 2 && ContainsWord(split, "timestamp") {
			return "time.Time"
		}
	}
	last := split[len(split)-1]
	if last == "string" {
		return "string"
	} else if last == "id" {
		return "string"
	}

	// just making sure
	for _, p := range []string{"two", "three", "four", "five"} {
		if strings.HasPrefix(split[0], p) {
			return ParseType(strings.Join(split[1:], " "))
		}
	}

	output := strings.Join(split, " ")
	output = strings.TrimPrefix(output, "partial ")
	output = strings.TrimPrefix(output, "a ")
	output = strings.TrimPrefix(output, "an ")
	output = strings.TrimSuffix(output, " object")

	return "*" + ConvertName(output)
}

func ContainsWord(arr []string, word string) bool {
	for _, w := range arr {
		if w == word {
			return true
		}
	}
	return false
}

func SplitTableRow(row string) []string {
	var split []string
	for _, t := range strings.Split(strings.Trim(row, "| "), " | ") {
		split = append(split, strings.TrimSpace(t))
	}
	return split
}

func RemoveHyperlinks(text string) string {
	return strings.TrimSpace(string(removeMarkdownURLRegexp.ReplaceAllString(text, "$1")))
}

func RemoveSuffix(text string, prefix string) string {
	return strings.TrimSpace(strings.TrimSuffix(text, prefix))
}

func ConvertWord(word string) string {
	if len(word) == 0 {
		return ""
	}

	switch strings.ToLower(word) {
	case "id":
		return "ID"
	case "ids":
		return "IDs"
	case "url":
		return "URL"
	case "urls":
		return "URLs"
	case "mfa":
		return "MFA"
	case "rpc":
		return "RPC"
	case "http":
		return "HTTP"
	case "https":
		return "HTTPS"
	case "http(s)":
		return "HTTP(S)"
	case "afk":
		return "AFK"
	case "nsfw":
		return "NSFW"
	default:
		return word
	}
}

func ConvertName(name string) string {
	var newName string

	//lint:ignore SA1019 strings.Title works fine
	for _, word := range strings.Split(strings.Title(strcase.ToDelimited(name, ' ')), " ") {
		newName += ConvertWord(word)
	}

	return newName
}

func ConvertDesc(desc string) string {
	var newDesc []string
	
	for _, word := range strings.Split(desc, " ") {
		newDesc = append(newDesc, strings.Trim(ConvertWord(word), "`"))
	}

	return strings.Join(append([]string{strings.ToUpper(string(newDesc[0][0])) + newDesc[0][1:]}, newDesc[1:]...), " ")
}
