package main

// email
// [separator]a-zA-Z0-9+@domain.com[separator]
// http[s]://   [separator]
/*
   } else if (! (is_wordchar(line[actual] + url_head) ||
     (ch == '-') || (ch == '_') || (ch == '\\') ||
     (ch == '.') || (ch == ':') || (ch == '/') ||
     (ch == '~') || (ch == '%') || (ch == '*') ||
     (ch == '$') || (ch == '[') || (ch == ']') ||
     (ch == '?') || (ch == '!') ||
     ((ch >= '0') && (ch <= '9')))) {
*/

import (
	"bytes"
	"flag"
	"github.com/client9/gospell"
	"github.com/client9/plaintext"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

var (
	stdout      *log.Logger // see below in init()
	defaultLog  *template.Template
	defaultWord *template.Template
)

const (
	defaultLogTmpl  = `{{ .Filename }}:{{ js .Original }}`
	defaultWordTmpl = `{{ .Original }}`
)

func init() {
	// we see it so it doesn't use a prefix or include a time stamp.
	stdout = log.New(os.Stdout, "", 0)
	defaultLog = template.Must(template.New("defaultLog").Parse(defaultLogTmpl))
	defaultWord = template.Must(template.New("defaultWord").Parse(defaultWordTmpl))
}

type diff struct {
	Filename string
	Path     string
	Original string
}

// This needs auditing as I believe it is wrong
func enURLChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' ||
		c == '_' ||
		c == '\\' ||
		c == '.' ||
		c == ':' ||
		c == ';' ||
		c == '/' ||
		c == '~' ||
		c == '%' ||
		c == '*' ||
		c == '$' ||
		c == '[' ||
		c == ']' ||
		c == '?' ||
		c == '#' ||
		c == '!'
}
func enNotURLChar(c rune) bool {
	return !enURLChar(c)
}

func removeURL(s string) string {
	var idx int

	for {
		if idx = strings.Index(s, "http"); idx == -1 {
			return s
		}

		news := s[:idx]
		endx := strings.IndexFunc(s[idx:], enNotURLChar)
		if endx != -1 {
			news = news + " " + s[idx+endx:]
		}
		s = news
	}
}

func main() {
	format := flag.String("f", "", "use Golang template for log message")
	flag.Parse()
	args := flag.Args()

	if len(*format) > 0 {
		t, err := template.New("custom").Parse(*format)
		if err != nil {
			log.Fatalf("Unable to compile log format: %s", err)
		}
		defaultLog = t
	}

	aff := "/usr/local/share/hunspell/en_US.aff"
	dic := "/usr/local/share/hunspell/en_US.dic"
	timeStart := time.Now()
	h, err := gospell.NewGoSpell(aff, dic)
	timeEnd := time.Now()

	// note: 10x too slow
	log.Printf("Loaded in %v", timeEnd.Sub(timeStart))

	if err != nil {
		log.Fatalf("%s", err)
	}

	splitter := gospell.NewSplitter(h.WordChars)
	//splitter := gospell.NewDelimiterSplitter()
	// stdin support
	if len(args) == 0 {
		raw, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Unable to read Stdin: %s", err)
		}
		raw = plaintext.StripTemplate(raw)
		md, err := plaintext.ExtractorByFilename("stdin")
		if err != nil {
			log.Fatalf("Unable to create parser: %s", err)
		}

		// extract plain text
		raw = md.Text(raw)

		// do character conversion "smart quotes" to quotes, etc
		// as specified in the Affix file
		rawstring := h.InputConversion(raw)

		// zap URLS
		s := removeURL(rawstring)

		// now get words
		words := splitter.Split(s)
		for _, word := range words {
			if known := h.Spell(word); !known {
				var output bytes.Buffer
				defaultLog.Execute(&output, diff{
					Filename: "stdin",
					Original: word,
				})
				// goroutine-safe print to os.Stdout
				stdout.Println(output.String())
			}
		}
	}
	for _, arg := range args {
		raw, err := ioutil.ReadFile(arg)
		if err != nil {
			log.Fatalf("Unable to read %q: %s", arg, err)
		}
		md, err := plaintext.ExtractorByFilename(arg)
		if err != nil {
			log.Fatalf("Unable to create parser: %s", err)
		}
		raw = plaintext.StripTemplate(raw)
		rawstring := string(md.Text(raw))
		s := removeURL(rawstring)
		words := splitter.Split(s)
		for _, word := range words {
			if known := h.Spell(word); !known {
				var output bytes.Buffer
				defaultLog.Execute(&output, diff{
					Filename: filepath.Base(arg),
					Path:     arg,
					Original: word,
				})
				// goroutine-safe print to os.Stdout
				stdout.Println(output.String())
			}
		}
	}
}
