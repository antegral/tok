// Package cmd holds the Cobra command tree for the tok CLI.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/antegral/tok/internal/input"
	"github.com/antegral/tok/internal/models"
	"github.com/antegral/tok/internal/provider"
)

// rootCmd is the single user-facing command: `tok <model> <file|->`.
//
// SilenceUsage and SilenceErrors are both true so a RunE error becomes the
// only thing printed to stderr — no double-printing of the error or
// auto-generated usage text. main.go owns the final stderr formatting.
var rootCmd = &cobra.Command{
	Use:           "tok <model> <file|->",
	Short:         "Count tokens for various LLMs",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// First arg (model spec) gets custom completion; second arg (file)
		// falls back to the shell's default file completion.
		if len(args) >= 1 {
			return nil, cobra.ShellCompDirectiveDefault
		}
		return models.CompleteSpec(toComplete)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		p, model, err := provider.Resolve(args[0])
		if err != nil {
			return err
		}
		text, err := input.Read(args[1])
		if err != nil {
			return err
		}
		count, err := p.Count(cmd.Context(), model, text)
		if err != nil {
			return err
		}
		fmt.Println(count)
		return nil
	},
}

// Execute runs the root command. main.go calls this and handles the error.
func Execute() error {
	return rootCmd.Execute()
}
