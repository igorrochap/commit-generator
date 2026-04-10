package generator

import (
	"bytes"
	"commit_generator/internal/loading"
	"commit_generator/internal/prompts"
	"fmt"
	"os/exec"
	"text/template"
)

func GenerateCommit(lang string) error {
	prompt, ok := prompts.Get(lang)
	if ok == false {
		return fmt.Errorf("language %s not supported", lang)
	}
	diff, err := GetDiff()
	if err != nil {
		return err
	}
	tmpl, err := template.New("prompt").Parse(prompt)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]string{"Diff": diff})
	if err != nil {
		return err
	}
	cmd := exec.Command("ollama", "run", "glm-5:cloud", "--hidethinking")
	cmd.Stdin = &buf

	done := make(chan struct{})

	loading.Start(done)
	out, err := cmd.Output()
	close(done)

	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
