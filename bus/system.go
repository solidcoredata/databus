package bus

import (
	"context"
)

/*
   Input reads the data bus and outputs a Bus or an error.
   Input can read previous versions of the bus as well.
   Input needs to be able to list other versions as well.

   Output persists to disk the full bus or delta Bus. Need to specify
   the version of the bus as well.

   First Implementation:
       A list of executables along with executable specific options are configured.
       The system calls each executable with a specific flag to get meta information:
           What node types it needs.
           (maybe a node type configuration as well for configuring the setup?)
       The second call passes:
           The full bus, the delta bus, the VersionDelta, and the meta-configuration for the output.

   Second Implementation:
       The list of runners can be an HTTP endpoint.
       The endpoint is hit that gives a list of node types it needs along with any other info.
       The second call passes the full buss, delta bus, and version delta, and meta-configuration.

   In both of these implementations, the runner should return a version number, to allow caching the
   first call. If there is a cached first call, then the second call may simply pass up the information
   along with the call version, and if the version is outdated, the first call may be called again and re-cached.

    The sysfs version should use the following output directories:
        /src (input)
        /version (output)
        /run (runner)
            /runner-name.com/sql
            /runner-name.com/ui
*/

type RunnerSetup struct {
	// This could be a filesystem root or an HTTP root path.
	Root     string
	Enteries []RunnerEntry
}
type RunnerEntry struct {
	Call    string
	Options map[string]string
}

type RunnerCall1Req struct {
	Options map[string]string
}
type RunnerCall1Resp struct {
	CallVersion int64
	Errors      []string

	NodeTypes []string
}

type RunnerCall2Req struct {
	CallVersion int64
	Options     map[string]string

	VersionDelta VersionDelta
	Bus          *Bus
	DeltaBus     interface{}
}
type RunnerCall2Resp struct {
	CallVersion int64
	Errors      []string
}

type System struct {
	Input
	Output
	Runner
}

type InputOptions struct {
	// Read the bus from src.
	Src bool

	// Zero for "current" version, -1 from current version, any positive
	// number for the exact version. Versions start at 1.
	// Ignored if Src is true.
	Version int64
}

type VersionDelta struct {
	Previous Version
	Current  Version
}

type Version struct {
	Version int64
}

type Input interface {
	ReadBus(ctx context.Context, opts InputOptions) (*Bus, error)
	ListVersion(ctx context.Context) ([]Version, error)
}

type Output interface {
	WriteBus(ctx context.Context, ver Version, currentBus *Bus) error
	WriteDelta(ctx context.Context, vdelta VersionDelta, deltaBus interface{}) error
}

type Runner interface {
	RunnerSetup(ctx context.Context) (RunnerSetup, error)
	Run(ctx context.Context, setup RunnerSetup, vdelta VersionDelta, currentBus *Bus, deltaBus interface{}) error
}
