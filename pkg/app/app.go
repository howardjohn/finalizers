package app

import (
	"os"
	"path"

	"github.com/ibuildthecloud/finalizers/pkg/filter"
	"github.com/ibuildthecloud/finalizers/pkg/world"
	cli "github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func New() *cobra.Command {
	root := cli.Command(&App{}, cobra.Command{
		Use:  path.Base(os.Args[0]),
		Long: "Stupid Finalizers",
	})
	return root
}

type App struct {
	Namespace  string `usage:"namespace" short:"n" env:"NAMESPACE"`
	All        bool   `usage:"print all objects with finalizers" short:"a"`
	Quiet      bool   `usage:"only print IDs" short:"q"`
	Output     string `usage:"yaml/json" short:"o"`
	Kubeconfig string `usage:"Location of kubeconfig" env:"KUBECONFIG"`
	Context    string `usage:"Context to use" env:"CONTEXT"`
}

func (a *App) Run(cmd *cobra.Command, args []string) error {
	clientConfig := kubeconfig.GetClientConfigWithContext(a.Kubeconfig, a.Context, os.Stdin)
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	trip, err := world.NewTrip(restConfig, &world.Options{
		Namespace: a.Namespace,
	})
	if err != nil {
		return err
	}

	filters := filter.Filters{
		filter.HasFinalizer,
	}
	if !a.All {
		filters = append(filters, filter.IsDeleted)
	}

	w := table.NewWriter([][]string{
		{"NAMESPACE", `{{default "" .Object.metadata.namespace}}`},
		{"NAME", "Object.metadata.name"},
		{"APIVERSION", "Object.apiVersion"},
		{"KIND", "Object.kind"},
		{"FINALIZERS", "Object.metadata.finalizers"},
	}, a.Namespace, a.Quiet, a.Output)
	w.AddFormatFunc("empty", func(obj interface{}) string {
		return convert.ToString(obj)
	})

	filters = append(filters, func(obj runtime.Object) runtime.Object {
		w.Write(obj)
		return obj
	})

	if err := trip.Wander(cmd.Context(), filters); err != nil {
		return err
	}

	return w.Close()
}
