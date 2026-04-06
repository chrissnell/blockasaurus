package winservice

import (
	"context"

	"golang.org/x/sys/windows/svc"
)

const serviceName = "blockasaurus"

// IsService reports whether the process is running as a Windows service.
func IsService() bool {
	ok, _ := svc.IsWindowsService()
	return ok
}

// RunFunc is the server's main blocking function. It receives a context
// that is cancelled when the service is asked to stop.
type RunFunc func(ctx context.Context) error

// Run registers the process with the Windows Service Control Manager
// and blocks until the service is stopped.
func Run(run RunFunc) error {
	return svc.Run(serviceName, &handler{run: run})
}

type handler struct {
	run RunFunc
}

func (h *handler) Execute(_ []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown

	s <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- h.run(ctx) }()

	s <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case err := <-errCh:
			s <- svc.Status{State: svc.StopPending}
			if err != nil {
				return true, 1
			}
			return false, 0

		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				s <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending}
				cancel()
				<-errCh // wait for clean exit
				return false, 0
			}
		}
	}
}
