package engine

type Node struct {
	Name   string
	Type   string
	Addr   string
	Delay  int // ms, -1 means untested
	Active bool
}

type Status struct {
	Running       bool
	Mode          string
	HTTPPort      int
	SOCKSPort     int
	MixedPort     int
	UploadSpeed   string
	DownloadSpeed string
	TotalUpload   string
	TotalDownload string
	Connections   int
}

type Kernel interface {
	Start() error
	Stop() error
	Restart() error
	Status() Status
	Nodes() []Node
	SetMode(mode string) error
	ReloadConfig() error
	TestNodeDelay(name string) (int, error)
	TestAllDelay() error
}
