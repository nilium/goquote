// Command goquote reads from a string standard input and prints it out as a quoted string for use in Go source code.
//
// goquote accepts an optional format specifier as its first and only argument.
// Formats are described in the command's usage text (-h or -help).
//
// This tool is primarily intended for use in editors.
//
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func usage() {
	fmt.Fprint(os.Stderr, `Usage: goquote [OPTIONS] [MODE [ARGS...]]

If no ARGS are given, standard input is read and written as a Go string
using a mode below.

MODE may be one of the following to change quote behavior:
  q   - Quoted string (default)
        "string"
  qa  - Quoted ASCII string
        "string\tescaped"
  ra  - Backquoted single-line ASCII string
        `+"`string`"+`
  r   - Backquoted single-line string
        `+"`string`"+`
  x   - Quoted byte string (\xHH only)
        "\x73\x74\x72\x69\x6e\x67"
  bs  - Quoted []byte() slice
        []byte("string")
  bsa - Quoted ASCII []byte() slice
        []byte("string")
  b   - Byte slice of octets
        []byte{0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x1}
  0b  - Byte slice of octets (with leading zero)
        []byte{0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x01}
  ba  - ASCII [N]byte array
        [6]byte{0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x1}
  0ba - ASCII [N]byte array (with leading zero)
        [6]byte{0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x01}
  j   - JSON string
        "string"

MODEs beginning with a 0 are equivalent to those that do not, except
that they render single-nibble bytes with a leading 0 (0x0f).

OPTIONS
  -s SEP        Separator (allows escape characters; default: "\n")
  -c            Trim trailing newline from standard input
  -h, -help     Print this usage text.
`,
	)
}

func write(buf *bytes.Buffer, b []byte, mode string) {
	var (
		lenstr = ""
		pad    = false
		bsmode = "q"
	)

loop:
	switch mode {
	case "", "q":
		buf.WriteString(strconv.Quote(string(b)))
	case "qa":
		buf.WriteString(strconv.QuoteToASCII(string(b)))
	case "ra":
		bsmode = "qa"
		fallthrough
	case "r":
		if !strconv.CanBackquote(string(b)) {
			mode = bsmode
			goto loop
		}
		buf.WriteByte('`')
		buf.Write(b)
		buf.WriteByte('`')
	case "x":
		buf.WriteByte('"')
		for _, c := range b {
			buf.WriteString(`\x`)
			h := strconv.FormatUint(uint64(c), 16)
			if len(h) == 1 {
				buf.WriteByte('0')
			}
			buf.WriteString(h)
		}
		buf.WriteByte('"')

	case "bsa":
		bsmode = "qa"
		fallthrough
	case "bs":
		buf.WriteString("[]byte(")
		write(buf, b, bsmode)
		buf.WriteByte(')')

	case "0ba":
		pad = true
		fallthrough
	case "ba":
		lenstr = strconv.Itoa(len(b))
		mode = "b"
		goto loop

	case "0b":
		pad = true
		fallthrough
	case "b":
		buf.WriteString("[" + lenstr + "]byte{")
		seenFirst := false
		for _, c := range b {
			if seenFirst {
				buf.WriteString(", ")
			}
			seenFirst = true
			buf.WriteString("0x")
			h := strconv.FormatUint(uint64(c), 16)
			if pad && len(h) < 2 {
				buf.WriteByte('0')
			}
			buf.WriteString(h)
		}
		buf.WriteByte('}')
	case "j": // JSON
		p, err := json.Marshal(string(b))
		if err != nil {
			log.Fatalf("unable to marshal %q as JSON: %v", b, err)
		}
		buf.Write(p)
	default:
		log.Fatalf("invalid format code %q", flag.Arg(0))
	}
}

func main() {
	sep := "\n"
	chomp := false
	flag.CommandLine.Usage = usage
	flag.StringVar(&sep, "s", sep, "Separator")
	flag.BoolVar(&chomp, "c", chomp, "Chomp")
	flag.Parse()

	if sep == `\0` {
		sep = "\x00"
	} else if u, err := strconv.Unquote(`"` + sep + `"`); err == nil {
		sep = u
	}

	mode := ""
	argv := flag.Args()
	if len(argv) > 0 {
		mode, argv = argv[0], argv[1:]
	}

	var buf bytes.Buffer
	if len(argv) == 0 {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		if n := len(b); chomp && n > 0 && b[n-1] == '\n' {
			b = b[:n-1]
		}
		write(&buf, b, mode)
	} else {
		for i, arg := range argv {
			if i > 0 {
				buf.WriteString(sep)
			}
			write(&buf, []byte(arg), mode)
		}
	}

	if sep == "\n" && isTTY() {
		buf.WriteString(sep)
	}

	var err error

	if err == nil && buf.Len() > 0 {
		_, err = buf.WriteTo(os.Stdout)
	}

	if err != nil {
		log.Fatal("Unable to write output string: ", err)
	}
}

// isTTY attempts to determine whether the current stdout refers to a terminal.
func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeNamedPipe) != os.ModeNamedPipe
}
