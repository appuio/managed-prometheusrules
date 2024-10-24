package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/appuio/managed-prometheusrules/cmd"
)

const (
	textMetricsAddr = `The address the metrics endpoint binds to.
Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.`
	textProbeAddr      = `The address the probe endpoint binds to.`
	textEnableElection = `Enable leader election for controller manager.
Enabling this will ensure there is only one active controller manager.`
	textSecureMetrics = `If set, the metrics endpoint is served securely via HTTPS.
Use --metrics-secure=false to use HTTP instead.`
	textEnableHTTP2      = `If set, HTTP/2 will be enabled for the metrics and webhook servers`
	textManagedNamespace = `The name of the namespace to manage PrometheusRule resources.`
	textWatchNamespaces  = `The name of a namespace to watch for PrometheusRule resources.`
	textWatchRegex       = `A regex for namespaces to watch for PrometheusRule resources.`
	textDryRun           = `Run mutating operations without perstisting.`
	textExternalParser   = `Path to external jsonnet file.
The external jsonnet file must output a valid PrometheusRuleSpec.`
	textExternalParams = `Path to external yaml file.
The external parameters file will be loaded as ext-code named 'params'.`
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "managed-prometheusrules",
	Short: "Manages PrometheuRules on a cluster.",
	Long: `This application watches for PrometheusRule resources.
This resources will be filtered and patched according
to the provided configuration. From the filtered and
patched resource a new PrometheuRule will be created.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: cmd.Root,
	// Run: func(cmd *cobra.Command, args []string) {
	// 	commands.Root(config)
	// },
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().String("metrics-address", "0", textMetricsAddr)
	RootCmd.PersistentFlags().String("probe-address", ":8081", textProbeAddr)
	RootCmd.PersistentFlags().Bool("metrics-secure", true, textSecureMetrics)
	RootCmd.PersistentFlags().Bool("enable-election", false, textEnableElection)
	RootCmd.PersistentFlags().Bool("enable-http2", false, textEnableHTTP2)
	RootCmd.PersistentFlags().String("managed-namespace", "default", textManagedNamespace)
	RootCmd.PersistentFlags().StringSlice("watch-namespace", []string{""}, textWatchNamespaces)
	RootCmd.PersistentFlags().String("watch-regex", "", textWatchRegex)
	RootCmd.PersistentFlags().Bool("dry-run", false, textDryRun)
	RootCmd.PersistentFlags().String("external-parser", "", textExternalParser)
	RootCmd.PersistentFlags().String("external-params", "", textExternalParams)

	viper.BindPFlag("metrics-address", RootCmd.PersistentFlags().Lookup("metrics-address"))
	viper.BindPFlag("probe-address", RootCmd.PersistentFlags().Lookup("probe-address"))
	viper.BindPFlag("metrics-secure", RootCmd.PersistentFlags().Lookup("metrics-secure"))
	viper.BindPFlag("enable-election", RootCmd.PersistentFlags().Lookup("enable-election"))
	viper.BindPFlag("enable-http2", RootCmd.PersistentFlags().Lookup("enable-http2"))
	viper.BindPFlag("managed-namespace", RootCmd.PersistentFlags().Lookup("managed-namespace"))
	viper.BindPFlag("watch-namespace", RootCmd.PersistentFlags().Lookup("watch-namespace"))
	viper.BindPFlag("watch-regex", RootCmd.PersistentFlags().Lookup("watch-regex"))
	viper.BindPFlag("dry-run", RootCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("external-parser", RootCmd.PersistentFlags().Lookup("external-parser"))
	viper.BindPFlag("external-params", RootCmd.PersistentFlags().Lookup("external-params"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
