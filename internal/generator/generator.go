package generator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/igorrochap/commitgen/internal/loading"
	"github.com/igorrochap/commitgen/internal/prompts"
	"github.com/igorrochap/commitgen/internal/selection"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\r`)

var (
	lineEndWord   = regexp.MustCompile(`([\p{L}\p{N}_]+)[ \t]*$`)
	lineStartWord = regexp.MustCompile(`^([\p{L}\p{N}_]+)`)
)

// unwrapLines joins soft line breaks within paragraphs.
// Mid-word duplicates (e.g. "internationalizatio\ninternationalization") are
// collapsed into the full word. Clean word-boundary wraps are joined with a
// space. Paragraph breaks (\n\n) and list item lines (-, *, +) are preserved.
func unwrapLines(s string) string {
	paragraphs := strings.Split(s, "\n\n")
	for i, p := range paragraphs {
		lines := strings.Split(p, "\n")
		if len(lines) <= 1 {
			continue
		}
		result := lines[0]
		for _, line := range lines[1:] {
			trimmed := strings.TrimLeft(line, " \t")
			if len(trimmed) > 0 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') {
				result += "\n" + line
				continue
			}
			endMatch := lineEndWord.FindStringSubmatch(result)
			startMatch := lineStartWord.FindStringSubmatch(trimmed)
			if len(endMatch) > 1 && len(startMatch) > 1 && strings.HasPrefix(startMatch[1], endMatch[1]) {
				result = result[:len(result)-len(endMatch[1])] + trimmed
			} else {
				result += " " + trimmed
			}
		}
		paragraphs[i] = result
	}
	return strings.Join(paragraphs, "\n\n")
}

type Options struct {
	Language string
	Model    string
}

func Run(option Options) error {
	prompt, err := getPrompt(option.Language)
	if err != nil {
		return err
	}
	diff, err := GetDiff()
	if err != nil {
		return err
	}
	tmpl, err := template.New("prompt").Parse(prompt)
	if err != nil {
		return err
	}
	err = selectOption(tmpl, diff, option.Model)
	return err
}

func getPrompt(language string) (string, error) {
	prompt, ok := prompts.Get(language)
	if ok == false {
		return "", fmt.Errorf("language %s not supported", language)
	}
	return prompt, nil
}

func selectOption(tmpl *template.Template, diff string, model string) error {
	end := false
	for end == false {
		commit, err := generateCommit(tmpl, diff, model)
		if err != nil {
			return err
		}
		result, err := selection.Run(commit)
		if err != nil {
			return err
		}
		switch result.Choice {
		case selection.Accept:
			makeCommit(commit)
			end = true
		case selection.Edit:
			updatedCommit, err := edit(commit)
			if err != nil {
				return err
			}
			makeCommit(updatedCommit)
			end = true
		}
	}
	return nil
}

func generateCommit(tmpl *template.Template, diff string, model string) (string, error) {
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, map[string]string{"Diff": diff})
	if err != nil {
		return "", err
	}
	cmd := exec.Command("ollama", "run", model, "--hidethinking")
	cmd.Stdin = &buf

	done := make(chan struct{})

	wait := loading.Start(done)
	out, err := cmd.Output()
	close(done)
	wait()

	if err != nil {
		return "", err
	}
	clean := ansiEscape.ReplaceAllString(string(out), "")
	clean = unwrapLines(clean)
	return strings.TrimSpace(clean), nil
}

func makeCommit(commit string) error {
	commitCmd := exec.Command("git", "commit", "-m", commit)
	err := commitCmd.Run()
	if err != nil {
		return err
	}
	getIdCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	id, err := getIdCmd.Output()
	if err != nil {
		return err
	}
	fmt.Printf("Commit %s created\n", strings.TrimSpace(string(id)))
	return nil
}

func edit(commit string) (string, error) {
	tmp, err := os.CreateTemp("", "commit-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(commit); err != nil {
		return "", err
	}
	tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		if _, err := exec.LookPath("nano"); err == nil {
			editor = "nano"
		} else {
			editor = "vim"
		}
	}

	editCmd := exec.Command(editor, tmp.Name())
	editCmd.Stdin = os.Stdin
	editCmd.Stdout = os.Stdout
	editCmd.Stderr = os.Stderr
	if err := editCmd.Run(); err != nil {
		return "", nil
	}

	content, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(content)), nil
}
