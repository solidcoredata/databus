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

type Project struct {
	// This could be a filesystem root or an HTTP root path.
	Root     string
	Enteries []RunnerEntry
}
type RunnerEntry struct {
	Name    string
	Call    string
	Options map[string]string
}

type CallNodeTypesRequest struct {
	Type    string // NodeTypes
	Options map[string]string
}
type CallNodeTypesResponse struct {
	Errors []string

	CallVersion int64
	NodeTypes   []string
}

type CallRunRequest struct {
	Type    string // Run
	Options map[string]string

	CallVersion int64
	Root        string

	VersionDelta VersionDelta
	Bus          *Bus
	DeltaBus     interface{}
}
type CallRunResponse struct {
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

type DeltaBus struct {
	_ struct{}
}

type Version struct {
	Version int64
}

type Input interface {
	ReadBus(ctx context.Context, opts InputOptions) (*Bus, error)
	ListVersion(ctx context.Context) ([]Version, error)
	ReadProject(ctx context.Context) (Project, error)
}

type Output interface {
	WriteBus(ctx context.Context, ver Version, currentBus *Bus) error
	WriteDelta(ctx context.Context, vdelta VersionDelta, deltaBus DeltaBus) error
}

type Runner interface {
	Run(ctx context.Context, project Project, vdelta VersionDelta, currentBus *Bus, deltaBus DeltaBus) error
}
