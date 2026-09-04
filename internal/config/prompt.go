package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// AuthHeader builds the Authorization header value from the two mutually
// exclusive flags. Both empty yields an empty header: an unguarded server.
func AuthHeader(bearer, basic string) (string, error) {
	switch {
	case bearer != "" && basic != "":
		return "", &Error{Msg: "--bearer-token and --basic-auth are mutually exclusive"}
	case bearer != "":
		return "Bearer " + bearer, nil
	case basic != "":
		// The whole string is encoded, so a password may contain colons.
		if !strings.Contains(basic, ":") {
			return "", &Error{Msg: `--basic-auth must be "username:password"`}
		}
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(basic)), nil
	}
	return "", nil
}

// MaskHeader keeps the scheme and the last four characters so a profile can be
// shown without leaking the credential.
func MaskHeader(h string) string {
	if h == "" {
		return ""
	}
	scheme, cred, found := strings.Cut(h, " ")
	if !found {
		return "****"
	}
	if len(cred) <= 4 {
		return scheme + " ****"
	}
	return scheme + " ****" + cred[len(cred)-4:]
}

// Prompter asks the questions of `gridraw config` on a terminal.
type Prompter struct {
	In  *bufio.Reader
	Out io.Writer
}

// NewPrompter reads answers from in and writes questions to out.
func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{In: bufio.NewReader(in), Out: out}
}

// Ask prints the question with its default and returns the answer, or def when
// the line is empty.
func (p *Prompter) Ask(question, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(p.Out, "%s [%s]: ", question, def)
	} else {
		fmt.Fprintf(p.Out, "%s: ", question)
	}
	line, err := p.In.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return def, nil
	}
	return line, nil
}

// Choose asks question until the answer is one of options; def is returned on
// an empty line.
func (p *Prompter) Choose(question string, options []string, def string) (string, error) {
	for {
		got, err := p.Ask(fmt.Sprintf("%s (%s)", question, strings.Join(options, "/")), def)
		if err != nil {
			return "", err
		}
		if contains(options, got) {
			return got, nil
		}
		fmt.Fprintf(p.Out, "  %q is not one of %s\n", got, strings.Join(options, ", "))
	}
}
