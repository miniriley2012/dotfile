package cli

import (
	"os"

	"github.com/knoebber/dotfile/local"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

type editCommand struct {
	getStorage func() (*local.Storage, error)
	fileName   string
}

var errEditorEnvVarNotSet = errors.New("EDITOR environment variable must be set")

func (e *editCommand) run(ctx *kingpin.ParseContext) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return errEditorEnvVarNotSet
	}

	s, err := e.getStorage()
	if err != nil {
		return err
	}

	path, err := s.GetPath(e.fileName)
	if err != nil {
		return err
	}

	cmd := execCommand(editor, path)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func addEditSubCommandToApplication(app *kingpin.Application, gs func() (*local.Storage, error)) {
	ec := &editCommand{
		getStorage: gs,
	}
	c := app.Command("edit", "open a tracked file in $EDITOR").Action(ec.run)
	c.Arg("file-name", "the file to edit").Required().StringVar(&ec.fileName)
}
