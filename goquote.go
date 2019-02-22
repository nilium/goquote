// Command goquote reads from a string standard input and prints it out as a quoted string for use in Go source code.
//
// goquote accepts an optional format specifier as its first and only argument. The format can be
// one of:
//
//  - q   :: Print a quoted UTF-8 string. (Default)
//  - r   :: Attempt to print a backquoted string. If unavailable, fall back to a normal quoted
//           string.
//  - ra  :: Attempt to print a backquoted string. Fallback to an ASCII-friendly quoted string.
//  - qa  :: Print a quoted ASCII string. Unicode values are escaped.
//  - x   :: Print a string made up of only escaped hex codes.
//  - b   :: Print a byte slice.
//  - 0b  :: Print a byte slice -- each octet is zero-padded.
//  - ba  :: Print a byte array.
//  - 0ba :: Print a byte array -- each octet is zero-padded.
//  - bs  :: Print a string-to-byte slice conversion ([]byte("quote")).
//  - bsa :: Print a string-to-byte slice conversion ([]byte("quote")). The quoted string only
//           contains ASCII characters.
//
// This tool is primarily intended for use in editors.
//
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func usage() {
	fmt.Fprint(os.Stderr, `Usage: goquote [OPTIONS] [MODE [ARGS...]]

If no ARGS are given, stdin is read and written as a Go string using
a mode below.

MODE may be one of the following to change quote behavior:
  q   - Quoted string. (Default)
  qa  - Quoted ASCII string.
  ra  - Backquoted single-line ASCII string.
  r   - Backquoted single-line string.
  x   - Quoted byte string (\xHH only).
  bs  - Quoted []byte() slice.
  bsa - Quoted ASCII []byte() slice.
  b   - Byte slice of octets.
  0b  - Byte slice of octets. (0-prefix)
  ba  - ASCII [N]byte array.
  0ba - ASCII [N]byte array. (0-prefix)

MODEs beginning with a 0 are equivalent to those that do not, except
that they render single-nibble bytes with a leading 0 (0x0f).

OPTIONS
  -s SEP   Separator (allows escape characters; default: "\n")
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
	case "", "q":
		buf.WriteString(strconv.Quote(string(b)))
	case "qa":
		buf.WriteString(strconv.QuoteToASCII(string(b)))
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

	case "ba":
		lenstr = strconv.Itoa(len(b))
		mode = "b"
		goto loop
	case "0ba":
		pad = true
		mode = "ba"
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
	default:
		log.Fatalf("invalid format code %q", flag.Arg(0))
	}
}

func main() {
	var sep = "\n"
	flag.CommandLine.Usage = usage
	flag.StringVar(&sep, "s", sep, "Separator")
	flag.Parse()

	if u, err := strconv.Unquote(`"` + sep + `"`); err == nil {
		sep = u
	}

	mode := ""
	if flag.NArg() > 0 {
		mode = flag.Arg(0)
	}

	var buf bytes.Buffer
	if flag.NArg() <= 1 {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		write(&buf, b, mode)
	} else {
		for i, arg := range flag.Args()[1:] {
			if i > 0 {
				buf.WriteString(sep)
			}
			write(&buf, []byte(arg), mode)
		}
	}

	var err error

	if err == nil && buf.Len() > 0 {
		_, err = buf.WriteTo(os.Stdout)
	}

	if err != nil {
		log.Fatal("Unable to write output string: ", err)
	}
}
