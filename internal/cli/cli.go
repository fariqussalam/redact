package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/fariqussalam/redact/internal/config"
	"github.com/fariqussalam/redact/internal/redactor"
)

// CLI defines the redact command-line interface.
type CLI struct {
	Config        string           `help:"Path to config file." type:"path"`
	Buffered      bool             `help:"Buffer output instead of flushing each line."`
	ConfigPath    ConfigPathCmd    `cmd:"" name:"config-path" help:"Print config path."`
	List          ListCmd          `cmd:"" help:"List active rules."`
	Clip          ClipCmd          `cmd:"" help:"Redact clipboard contents and copy result back."`
	AddName       AddNameCmd       `cmd:"" name:"add-name" help:"Add an exact name rule for fields and URL params."`
	RemoveName    RemoveNameCmd    `cmd:"" name:"remove-name" help:"Remove an exact name rule."`
	AddPattern    AddPatternCmd    `cmd:"" name:"add-pattern" help:"Add a regex name rule for fields and URL params."`
	RemovePattern RemovePatternCmd `cmd:"" name:"remove-pattern" help:"Remove a regex name rule."`
}

// ConfigPathCmd prints the active config path.
type ConfigPathCmd struct{}

// ClipCmd redacts clipboard contents and writes the redacted result back.
type ClipCmd struct {
	Print bool `help:"Also print redacted clipboard contents to stdout."`
}

// ListCmd prints configured and default rules.
type ListCmd struct {
	User     bool `help:"Show user rules only."`
	Defaults bool `help:"Show default rules only."`
}

// AddNameCmd adds an exact name rule for both field and URL-parameter contexts.
type AddNameCmd struct {
	Name string `arg:""`
}

// RemoveNameCmd removes an exact name rule.
type RemoveNameCmd struct {
	Name string `arg:""`
}

// AddPatternCmd adds a regex name rule for both field and URL-parameter contexts.
type AddPatternCmd struct {
	Pattern string `arg:""`
}

// RemovePatternCmd removes a regex name rule.
type RemovePatternCmd struct {
	Pattern string `arg:""`
}

type ctx struct {
	path     string
	explicit bool
}

// Main runs the CLI with args and returns a process exit code.
func Main(args []string) int {
	var cli CLI
	parser, err := kong.New(&cli, kong.Name("redact"), kong.Description("Local streaming text redactor."))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if firstCommand(args) == "" && !hasHelp(args) {
		cfgPath := flagConfig(args)
		if isStdinPiped() {
			return runStream(cfgPath != "", cfgPath, flagBuffered(args))
		}
		fmt.Fprintln(os.Stderr, "usage: redact [--config PATH] <command>\n       command | redact")
		return 0
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	path, explicit, err := configPath(cli.Config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	err = kctx.Run(&ctx{path, explicit})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func (ConfigPathCmd) Run(c *ctx) error { fmt.Println(c.path); return nil }
func (cmd ClipCmd) Run(c *ctx) error {
	in, err := readClipboard()
	if err != nil {
		return err
	}
	var out strings.Builder
	if err := streamCommand(c, strings.NewReader(in), &out); err != nil {
		return err
	}
	if err := writeClipboard(out.String()); err != nil {
		return err
	}
	if cmd.Print {
		fmt.Print(out.String())
	} else {
		fmt.Fprintln(os.Stderr, "redacted clipboard")
	}
	return nil
}
func (cmd ListCmd) Run(c *ctx) error {
	cfg, exists, err := config.Load(c.path, c.explicit)
	if err != nil {
		return err
	}
	e := config.EffectiveRules(cfg)
	if !exists {
		fmt.Fprintf(os.Stdout, "Config: %s (not created)\n\n", c.path)
	}
	printRules("Fields", e.Fields, cmd)
	printRules("Field patterns", e.FieldPatterns, cmd)
	printRules("URL params", e.URLParams, cmd)
	printRules("URL param patterns", e.URLParamPatterns, cmd)
	return nil
}
func (cmd AddNameCmd) Run(c *ctx) error       { return mutateName(c, true, cmd.Name) }
func (cmd RemoveNameCmd) Run(c *ctx) error    { return mutateName(c, false, cmd.Name) }
func (cmd AddPatternCmd) Run(c *ctx) error    { return mutatePattern(c, true, cmd.Pattern) }
func (cmd RemovePatternCmd) Run(c *ctx) error { return mutatePattern(c, false, cmd.Pattern) }

func firstCommand(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" || args[i] == "-config" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--config=") {
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			continue
		}
		return args[i]
	}
	return ""
}

func flagConfig(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(args[i], "--config=") {
			return strings.TrimPrefix(args[i], "--config=")
		}
	}
	return ""
}

func flagBuffered(args []string) bool {
	for _, arg := range args {
		if arg == "--buffered" {
			return true
		}
	}
	return false
}

func hasHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func mutateName(c *ctx, add bool, name string) error {
	name, err := config.NormalizeName(name)
	if err != nil {
		return err
	}
	if add {
		cfg, _, err := config.Load(c.path, false)
		if err != nil {
			return err
		}
		effective := config.EffectiveRules(cfg)
		if config.ContainsRule(effective.Fields, name) && config.ContainsRule(effective.URLParams, name) {
			fmt.Println("name already covered:", name)
			return nil
		}
		changed, err := config.AddListValue(c.path, "names", name)
		if err != nil {
			return err
		}
		if changed {
			fmt.Println("added:", name)
		} else {
			fmt.Println("already present:", name)
		}
		return nil
	}
	changed, err := config.RemoveListValue(c.path, "names", name)
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("removed:", name)
	} else {
		fmt.Println("not present:", name)
	}
	return nil
}

func mutatePattern(c *ctx, add bool, pattern string) error {
	if _, err := redactor.New(config.Effective{Mask: config.DefaultMask, FieldPatterns: []config.Rule{{Value: pattern}}, URLParamPatterns: []config.Rule{{Value: pattern}}}); err != nil {
		return err
	}
	if add {
		changed, err := config.AddListValue(c.path, "patterns", pattern)
		if err != nil {
			return err
		}
		if changed {
			fmt.Println("added:", pattern)
		} else {
			fmt.Println("already present:", pattern)
		}
		return nil
	}
	changed, err := config.RemoveListValue(c.path, "patterns", pattern)
	if err != nil {
		return err
	}
	if changed {
		fmt.Println("removed:", pattern)
	} else {
		fmt.Println("not present:", pattern)
	}
	return nil
}

func runStream(explicit bool, path string, buffered bool) int {
	p, ex, err := configPath(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := streamPath(p, explicit || ex, os.Stdin, os.Stdout, !buffered); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

func streamCommand(c *ctx, in io.Reader, out io.Writer) error {
	return streamPath(c.path, c.explicit, in, out, false)
}

func streamPath(path string, explicit bool, in io.Reader, out io.Writer, flushEachLine bool) error {
	cfg, _, err := config.Load(path, explicit)
	if err != nil {
		return err
	}
	r, err := redactor.New(config.EffectiveRules(cfg))
	if err != nil {
		return err
	}
	return stream(r, in, out, flushEachLine)
}

func stream(r *redactor.Redactor, in io.Reader, out io.Writer, flushEachLine bool) error {
	br := bufio.NewReader(in)
	bw := bufio.NewWriter(out)
	var dst []byte
	for {
		line, err := br.ReadBytes('\n')
		if len(line) > 0 {
			dst = r.RedactLine(dst[:0], line)
			if _, werr := bw.Write(dst); werr != nil {
				return fmt.Errorf("write redacted output: %w", werr)
			}
			if flushEachLine {
				if err := bw.Flush(); err != nil {
					return fmt.Errorf("flush redacted output: %w", err)
				}
			}
		}
		if err == io.EOF {
			if err := bw.Flush(); err != nil {
				return fmt.Errorf("flush redacted output: %w", err)
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
	}
}
func configPath(p string) (string, bool, error) {
	if p != "" {
		return p, true, nil
	}
	d, err := config.DefaultPath()
	return d, false, err
}
func isStdinPiped() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice == 0
}
func printRules(title string, rules []config.Rule, cmd ListCmd) {
	fmt.Println(title + ":")
	n := 0
	for _, r := range rules {
		if cmd.User && r.Source != "user" {
			continue
		}
		if cmd.Defaults && r.Source != "default" {
			continue
		}
		fmt.Printf("  %-28s %s\n", r.Value, r.Source)
		n++
	}
	if n == 0 {
		fmt.Println("  none")
	}
	fmt.Println()
}

func readClipboard() (string, error) {
	if _, err := exec.LookPath("pbpaste"); err == nil {
		out, err := exec.Command("pbpaste").Output()
		return string(out), err
	}
	if _, err := exec.LookPath("wl-paste"); err == nil {
		out, err := exec.Command("wl-paste", "--no-newline").Output()
		return string(out), err
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
		return string(out), err
	}
	return "", fmt.Errorf("no supported clipboard reader found (tried pbpaste, wl-paste, xclip)")
}

func writeClipboard(s string) error {
	if _, err := exec.LookPath("pbcopy"); err == nil {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(s)
		return cmd.Run()
	}
	if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command("wl-copy")
		cmd.Stdin = strings.NewReader(s)
		return cmd.Run()
	}
	if _, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command("xclip", "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(s)
		return cmd.Run()
	}
	return fmt.Errorf("no supported clipboard writer found (tried pbcopy, wl-copy, xclip)")
}
