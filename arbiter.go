// Arbiter runs system tests on the target system.
//
// By providing custom modules, Arbiter is able to generate traffic and monitor
// any system, measuing CPU, memory, metrics, and logs. Arbiter can judge a
// system based on those four parameters. Add rates for operations and
// thresholds to verify the software is staying within expected boundaries.
package arbiter

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"tres-bon.se/arbiter/pkg/module"
	modulearg "tres-bon.se/arbiter/pkg/module/arg"
	"tres-bon.se/arbiter/pkg/monitor"
	cpumonitor "tres-bon.se/arbiter/pkg/monitor/cpu"
	logmonitor "tres-bon.se/arbiter/pkg/monitor/log"
	memorymonitor "tres-bon.se/arbiter/pkg/monitor/memory"
	metricmonitor "tres-bon.se/arbiter/pkg/monitor/metric"
	"tres-bon.se/arbiter/pkg/report"
	"tres-bon.se/arbiter/pkg/traffic"
	"tres-bon.se/arbiter/pkg/zerologr"
)

const (
	FLAGSET_CLI  = modulearg.FLAGSET
	FLAGSET_GEN  = "generate"
	FLAGSET_FILE = "file"
)

var (
	duration time.Duration = time.Minute * 5

	subcommands     = []string{FLAGSET_CLI, FLAGSET_GEN, FLAGSET_FILE}
	subcommandIndex = -1

	startLogger = zerologr.New(&zerologr.Opts{Console: true, V: 10}).WithName("start")
)

// Runs the Arbiter. Blocks until SIGINT, SIGTERM or when the test duration
// runs out (5 minute default).
func Run(modules module.Modules) error {
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "%s [subcommand]\n\n", os.Args[0])
		fmt.Fprint(flag.CommandLine.Output(), "subcommands:\n")
		fmt.Fprint(flag.CommandLine.Output(), "  cli      Run using CLI flags.\n")
		fmt.Fprint(flag.CommandLine.Output(), "  generate Generate a test model file.\n")
		fmt.Fprint(flag.CommandLine.Output(), "  file     Run from a test model file.\n")
		fmt.Fprint(flag.CommandLine.Output(), "\n")
		fmt.Fprint(flag.CommandLine.Output(), "global flags:\n")
		flag.PrintDefaults()
	}

	// Global flags
	flag.DurationVar(&duration, "duration", duration, "The duration of the test run.")

	// To trigger on --help and parse global flags
	flag.Parse()

	// Verify a subcommand has been invoked.
	subcommandIndex = slices.IndexFunc(os.Args, func(e string) bool {
		return slices.Contains(subcommands, e)
	})
	if subcommandIndex == -1 {
		flag.CommandLine.SetOutput(os.Stderr)
		fmt.Fprint(flag.CommandLine.Output(), "no subcommand given\n")
		flag.Usage()
		os.Exit(1)
	}

	// Check invoked subcommand
	switch os.Args[subcommandIndex] {
	case modulearg.FLAGSET:
		return handleCli(modules)
	case FLAGSET_GEN:
		return handleGen(modules)
	case FLAGSET_FILE:
		return handleFile(modules)
	default:
		flag.CommandLine.SetOutput(os.Stderr)
		fmt.Fprintf(flag.CommandLine.Output(), "subcommand not found: %s\n", os.Args[1])
		flag.Usage()
		os.Exit(1)
	}

	return nil
}

// Handle the CLI subcommand call, registering cmd line flags for both module
// arguments and their operations and parsing them. Runs the modules with the
// input flags.
func handleCli(modules module.Modules) error {
	for _, m := range modules {
		for _, a := range m.Args() {
			modulearg.Register(m.Name(), a)
		}

		// Add operation args
		for _, op := range m.Ops() {
			modulearg.RegisterOp(m.Name(), op)
		}
	}

	modulearg.Parse(os.Args[subcommandIndex+1:])

	startLogger.Info("parsed CLI arguments")

	return run(modules)
}

// Handles the generate subcommand. Generates a test model file based on the
// input modules.
func handleGen(_ module.Modules) error {
	var output string
	fs := flag.NewFlagSet(FLAGSET_GEN, flag.ExitOnError)
	fs.StringVar(&output, "output", "./arbiter.yaml", "Output path for the generated test model file.")
	err := fs.Parse(os.Args[subcommandIndex+1:])
	if err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		os.Exit(1)
	}

	// TODO: generate using input modules
	panic("not implemented")
}

// Handles the file subcommand, parsing the input test model file and running
// the modules with the file's settings.
func handleFile(_ module.Modules) error {
	outputPath := "./arbiter.yaml"
	fs := flag.NewFlagSet(FLAGSET_FILE, flag.ExitOnError)
	fs.StringVar(&outputPath, "path", outputPath, "Path to a test model file.")
	err := fs.Parse(os.Args[subcommandIndex+1:])
	if err != nil {
		fs.SetOutput(os.Stderr)
		fs.Usage()
		os.Exit(1)
	}

	// TODO: parse and run from file
	panic("not implemented")
}

// Runs the input modules and starts generating traffic. Creates a traffic
// model based on the modules opertation settings. Aborts on SIGINT, SIGTERM
// or when the test duration runs out. Will immediately exit if any module
// returns an error from its call to Run().
func run(modules module.Modules) error {
	startLogger.Info("preparing to run the modules")

	// Start each module, exit on error
	for _, m := range modules {
		startLogger.Info("starting", "module", m.Name())
		if err := m.Run(); err != nil {
			startLogger.Error(err, "failed to start module", "module", m.Name())
			panic(err)
		}
	}

	startLogger.Info("all modules started")

	// Hook up monitor and reporter
	// TODO: add choice of reporter implementation - future, new arbiter.Run(...)
	reporter := &report.YAMLReporter{}
	// TODO: add threshold support
	// TODO: add toggleable monitor options
	// TODO: add choice of monitor implementations - future, new arbiter.Run(...)
	monitor := &monitor.Monitor{
		CPU:      cpumonitor.NewLocalCPUMonitor(0),
		Memory:   memorymonitor.NewLocalMemoryMonitor(0),
		Metric:   metricmonitor.NewMetricMonitor(),
		Log:      logmonitor.NewLogFileMonitor(),
		Reporter: reporter,
	}

	// Start traffic and monitor, with a deadline set to time.Now() + test duration
	startLogger.Info("starting the monitor and traffic generation")

	runCtx := context.Background()
	runCtx, deadlineStop := context.WithDeadline(runCtx, time.Now().Add(5*time.Second))
	defer deadlineStop()
	if err := monitor.Start(runCtx); err != nil {
		startLogger.Error(err, "failed to start the monitor")
		panic(err)
	}

	if err := traffic.Run(runCtx, modules, reporter); err != nil {
		startLogger.Error(err, "failed to start traffic")
		panic(err)
	}

	// Start signal interceptor for SIGINT and SIGTERM
	runCtx, signalStop := signal.NotifyContext(runCtx, syscall.SIGINT, syscall.SIGTERM)
	defer signalStop()

	startLogger.Info("awaiting stop signal")
	<-runCtx.Done()

	startLogger.Info("got stop signal")

	startLogger.Info("stopping traffic")
	traffic.AwaitStop()

	startLogger.Info("stopping modules")
	for _, m := range modules {
		if err := m.Stop(); err != nil {
			startLogger.Error(err, "module stop reported an error", "module", m.Name())
		}
	}

	startLogger.Info("finalising report")
	reporter.Finalise()

	return nil
}
