package cmd

import (
	"fmt"
	"net/http"

	"github.com/garethjevans/pr-controller/pkg/prcontroller/server"

	"github.com/spf13/cobra"
)

var (
	BindAddress string
	Port        int
)

// NewRunCmd creates a new run command.
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run the webserver",
		Long:    "",
		Example: "pr-controller run",
		Aliases: []string{"r"},
		RunE: func(cmd *cobra.Command, args []string) error {
			mux := http.NewServeMux()

			gh, err := server.NewWebHook("github")
			if err != nil {
				return err
			}

			mux.HandleFunc("/github", gh.Handle)

			gl, err := server.NewWebHook("gitlab")
			if err != nil {
				return err
			}
			mux.HandleFunc("/gitlab", gl.Handle)

			// FIXME should we handle more here?

			a := fmt.Sprintf("%s:%d", BindAddress, Port)
			fmt.Printf("listening on %s\n", a)

			return http.ListenAndServe(a, mux)
		},
		Args:         cobra.NoArgs,
		SilenceUsage: true,
	}

	cmd.Flags().StringVarP(&BindAddress, "bind-address", "", "localhost", "The address to bind to (default: localhost)")
	cmd.Flags().IntVarP(&Port, "port", "p", 8080, "The port to run the webserver on (default: 8080)")

	return cmd
}
