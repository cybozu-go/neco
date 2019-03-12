package placemat

import (
	"context"
	"net/http"
	"strings"

	"github.com/cybozu-go/placemat/web"
	"github.com/cybozu-go/well"
)

// Server is the API Server of placemat.
type Server struct {
	cluster *Cluster
	vms     map[string]*NodeVM
	runtime *Runtime
}

// NewServer creates a new Server instance.
func NewServer(cluster *Cluster, vms map[string]*NodeVM, r *Runtime) *Server {
	return &Server{
		cluster: cluster,
		vms:     vms,
		runtime: r,
	}
}

// Handler implements http.Handler
func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/nodes") {
		s.handleNodes(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/pods") {
		s.handlePods(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/networks") {
		s.handleNetworks(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/snapshots") {
		s.handleSnapshots(w, r)
		return
	}
	web.RenderError(r.Context(), w, web.APIErrBadRequest)
}

func (s Server) newNodeStatus(node *Node, vm *NodeVM) *web.NodeStatus {
	status := &web.NodeStatus{
		Name:      node.Name,
		Taps:      node.taps,
		CPU:       node.CPU,
		Memory:    node.Memory,
		UEFI:      node.UEFI,
		IsRunning: vm.IsRunning(),
	}
	status.SMBIOS.Serial = node.SMBIOS.Serial
	status.SMBIOS.Manufacturer = node.SMBIOS.Manufacturer
	status.SMBIOS.Product = node.SMBIOS.Product
	if !s.runtime.graphic {
		status.SocketPath = s.runtime.socketPath(node.Name)
	}
	status.Volumes = make([]string, len(node.Volumes))
	for i, v := range node.Volumes {
		status.Volumes[i] = v.Name
	}
	return status
}

func (s Server) newPodStatus(pod *Pod) *web.PodStatus {
	status := &web.PodStatus{
		Name:  pod.Name,
		Veths: pod.veths,
		PID:   pod.pid,
		UUID:  pod.uuid,
	}
	status.Volumes = make([]string, len(pod.Volumes))
	for i, v := range pod.Volumes {
		status.Volumes[i] = v.Name
	}
	status.Apps = make([]string, len(pod.Apps))
	for i, v := range pod.Apps {
		status.Apps[i] = v.Name
	}
	return status
}

func splitParams(path string) []string {
	paths := strings.Split(path, "/")
	var params []string
	for _, str := range paths {
		if str != "" {
			params = append(params, str)
		}
	}
	return params
}

func (s Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	params := splitParams(r.URL.Path)
	if r.Method == "GET" && len(params) == 1 {
		statuses := make([]*web.NodeStatus, len(s.cluster.Nodes))
		for i, node := range s.cluster.Nodes {
			statuses[i] = s.newNodeStatus(node, s.vms[node.SMBIOS.Serial])
		}
		web.RenderJSON(w, statuses, http.StatusOK)
	} else if r.Method == "GET" && len(params) == 2 {
		if node, ok := s.cluster.nodeMap[params[1]]; ok {
			status := s.newNodeStatus(node, s.vms[node.SMBIOS.Serial])
			web.RenderJSON(w, status, http.StatusOK)
		} else {
			web.RenderError(r.Context(), w, web.APIErrNotFound)
		}
	} else if r.Method == "POST" && len(params) == 3 {
		if node, ok := s.cluster.nodeMap[params[1]]; ok {
			switch params[2] {
			case "start":
				s.vms[node.SMBIOS.Serial].PowerOn()
			case "stop":
				s.vms[node.SMBIOS.Serial].PowerOff()
			case "restart":
				s.vms[node.SMBIOS.Serial].PowerOff()
				s.vms[node.SMBIOS.Serial].PowerOn()
			default:
				web.RenderError(r.Context(), w, web.APIErrBadRequest)
				return
			}
			web.RenderJSON(w, s.vms[node.SMBIOS.Serial].IsRunning(), http.StatusOK)
		} else {
			web.RenderError(r.Context(), w, web.APIErrNotFound)
		}
	} else {
		web.RenderError(r.Context(), w, web.APIErrBadRequest)
	}
}

func (s Server) handlePods(w http.ResponseWriter, r *http.Request) {
	params := splitParams(r.URL.Path)
	if r.Method == "GET" && len(params) == 1 {
		statuses := make([]*web.PodStatus, len(s.cluster.Pods))
		for i, pod := range s.cluster.Pods {
			statuses[i] = s.newPodStatus(pod)
		}
		web.RenderJSON(w, statuses, http.StatusOK)
	} else if r.Method == "GET" && len(params) == 2 {
		if pod, ok := s.cluster.podMap[params[1]]; ok {
			status := s.newPodStatus(pod)
			web.RenderJSON(w, status, http.StatusOK)
		} else {
			web.RenderError(r.Context(), w, web.APIErrNotFound)
		}
		/* not working
		} else if r.Method == "POST" && len(params) == 3 {
			if pod, ok := s.cluster.podMap[params[1]]; ok {
				var cmds [][]string
				switch params[2] {
				case "start":
					cmds = append(cmds, []string{"ip", "netns", "exec", "pm_" + pod.Name, "rkt", "start", pod.uuid})
				case "stop":
					cmds = append(cmds, []string{"ip", "netns", "exec", "pm_" + pod.Name, "rkt", "stop", pod.uuid})
				case "restart":
				default:
					renderError(r.Context(), w, APIErrBadRequest)
					return
				}
				err := execCommands(r.Context(), cmds)
				if err != nil {
					renderError(r.Context(), w, InternalServerError(err))
				} else {
					renderJSON(w, "ok", http.StatusOK)
				}
			} else {
				renderError(r.Context(), w, APIErrNotFound)
			}
		*/
	} else {
		web.RenderError(r.Context(), w, web.APIErrBadRequest)
	}
}

func (s Server) handleNetworks(w http.ResponseWriter, r *http.Request) {
	params := splitParams(r.URL.Path)

	if r.Method == "POST" && len(params) == 3 {
		var cmds [][]string
		switch params[2] {
		case "up":
			cmds = append(cmds, []string{"ip", "link", "set", "dev", params[1], "up"})
		case "down":
			cmds = append(cmds, []string{"ip", "link", "set", "dev", params[1], "down"})
		case "delay":
			delay := r.URL.Query().Get("delay")
			if len(delay) == 0 {
				delay = "100ms"
			}
			cmds = append(cmds, []string{"tc", "qdisc", "add", "dev", params[1], "root", "netem", "delay", delay})
		case "loss":
			loss := r.URL.Query().Get("loss")
			if len(loss) == 0 {
				loss = "10%"
			}
			cmds = append(cmds, []string{"tc", "qdisc", "add", "dev", params[1], "root", "netem", "loss", loss})
		case "clear":
			cmds = append(cmds, []string{"tc", "qdisc", "del", "dev", params[1], "root"})
		default:
			web.RenderError(r.Context(), w, web.APIErrBadRequest)
			return
		}

		err := execCommands(r.Context(), cmds)
		if err != nil {
			web.RenderError(r.Context(), w, web.InternalServerError(err))
		} else {
			web.RenderJSON(w, "ok", http.StatusOK)
		}
	} else {
		web.RenderError(r.Context(), w, web.APIErrBadRequest)
	}
}

func (s Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	params := splitParams(r.URL.Path)
	if r.Method == "GET" && len(params) == 1 {
		list := make(map[string]interface{})
		for _, node := range s.cluster.Nodes {
			vm := s.vms[node.SMBIOS.Serial]
			out, err := vm.ListSnapshots(r.Context(), node)
			if err != nil {
				// Skip to show list of snapshots
				continue
			}
			list[node.Name] = out
		}
		web.RenderJSON(w, list, http.StatusOK)
	} else if r.Method == "POST" && len(params) == 3 {
		switch params[1] {
		case "save":
			err := s.saveSnapshot(r.Context(), params[2])
			if err != nil {
				web.RenderError(r.Context(), w, web.InternalServerError(err))
			}
		case "load":
			err := s.loadSnapshot(r.Context(), params[2])
			if err != nil {
				web.RenderError(r.Context(), w, web.InternalServerError(err))
			}
		default:
			web.RenderError(r.Context(), w, web.APIErrBadRequest)
		}
	} else {
		web.RenderError(r.Context(), w, web.APIErrBadRequest)
	}
}

func (s Server) saveSnapshot(ctx context.Context, name string) error {
	env := well.NewEnvironment(ctx)
	for _, node := range s.cluster.Nodes {
		vm := s.vms[node.SMBIOS.Serial]
		node := node
		env.Go(func(ctx context.Context) error {
			return vm.SaveVM(ctx, node, name)
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return err
	}
	return s.resumeVM(ctx)
}

func (s Server) loadSnapshot(ctx context.Context, name string) error {
	env := well.NewEnvironment(ctx)
	for _, node := range s.cluster.Nodes {
		vm := s.vms[node.SMBIOS.Serial]
		node := node
		env.Go(func(ctx context.Context) error {
			return vm.LoadVM(ctx, node, name)
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		return err
	}
	return s.resumeVM(ctx)
}

func (s Server) resumeVM(ctx context.Context) error {
	env := well.NewEnvironment(ctx)
	for _, node := range s.cluster.Nodes {
		vm := s.vms[node.SMBIOS.Serial]
		env.Go(func(ctx context.Context) error {
			return vm.ResumeVM(ctx)
		})
	}
	env.Stop()
	return env.Wait()
}
