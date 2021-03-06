package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/log"
	"github.com/containerd/containerd/progress"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

var imageCommand = cli.Command{
	Name:  "images",
	Usage: "image management",
	Subcommands: cli.Commands{
		imagesListCommand,
		imageRemoveCommand,
	},
}

var imagesListCommand = cli.Command{
	Name:        "list",
	Aliases:     []string{"ls"},
	Usage:       "list images known to containerd",
	ArgsUsage:   "[flags] <ref>",
	Description: `List images registered with containerd.`,
	Flags:       []cli.Flag{},
	Action: func(clicontext *cli.Context) error {
		ctx, cancel := appContext(clicontext)
		defer cancel()

		client, err := getClient(clicontext)
		if err != nil {
			return err
		}

		imageStore := client.ImageService()
		cs := client.ContentStore()

		images, err := imageStore.List(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to list images")
		}

		tw := tabwriter.NewWriter(os.Stdout, 1, 8, 1, ' ', 0)
		fmt.Fprintln(tw, "REF\tTYPE\tDIGEST\tSIZE\t")
		for _, image := range images {
			size, err := image.Size(ctx, cs)
			if err != nil {
				log.G(ctx).WithError(err).Errorf("failed calculating size for image %s", image.Name)
			}

			fmt.Fprintf(tw, "%v\t%v\t%v\t%v\t\n", image.Name, image.Target.MediaType, image.Target.Digest, progress.Bytes(size))
		}

		return tw.Flush()
	},
}

var imageRemoveCommand = cli.Command{
	Name:        "remove",
	Aliases:     []string{"rm"},
	Usage:       "Remove one or more images by reference.",
	ArgsUsage:   "[flags] <ref> [<ref>, ...]",
	Description: `Remove one or more images by reference.`,
	Flags:       []cli.Flag{},
	Action: func(clicontext *cli.Context) error {
		var (
			exitErr error
		)
		ctx, cancel := appContext(clicontext)
		defer cancel()

		client, err := getClient(clicontext)
		if err != nil {
			return err
		}

		imageStore := client.ImageService()

		for _, target := range clicontext.Args() {
			if err := imageStore.Delete(ctx, target); err != nil {
				if !errdefs.IsNotFound(err) {
					if exitErr == nil {
						exitErr = errors.Wrapf(err, "unable to delete %v", target)
					}
					log.G(ctx).WithError(err).Errorf("unable to delete %v", target)
					continue
				}
			}

			fmt.Println(target)
		}

		return exitErr
	},
}
