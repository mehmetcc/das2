package httpx

type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformMac     Platform = "mac"
	PlatformWindows Platform = "win"
	PlatformLinux   Platform = "linux"
	PlatformWeb     Platform = "web"
)

type DeviceMeta struct {
	DeviceID   string   `header:"X-Device-Id"      validate:"omitempty,min=8,max=128"` // allow UUID/ULID/custom
	DeviceName string   `header:"X-Device-Name"    validate:"omitempty,min=1,max=64"`  // human label
	Platform   Platform `header:"X-Client-Platform" validate:"omitempty,oneof=ios android mac win linux web"`
	AppVersion string   `header:"X-App-Version"    validate:"omitempty,min=1,max=32"` // optional semantic version
	UserAgent  string   `header:"-"                validate:"omitempty,max=256"`      // from r.UserAgent()
	IP         string   `header:"-"                validate:"omitempty,max=64"`       // derived from X-Forwarded-For/RemoteAddr
}
