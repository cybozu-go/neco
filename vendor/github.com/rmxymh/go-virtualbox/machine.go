package virtualbox

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type MachineState string

const (
	Poweroff = MachineState("poweroff")
	Running  = MachineState("running")
	Paused   = MachineState("paused")
	Saved    = MachineState("saved")
	Aborted  = MachineState("aborted")
)

type Flag int

// Flag names in lowercases to be consistent with VBoxManage options.
const (
	F_acpi Flag = 1 << iota
	F_ioapic
	F_rtcuseutc
	F_cpuhotplug
	F_pae
	F_longmode
	F_synthcpu
	F_hpet
	F_hwvirtex
	F_triplefaultreset
	F_nestedpaging
	F_largepages
	F_vtxvpid
	F_vtxux
	F_accelerate3d
)

// Convert bool to "on"/"off"
func bool2string(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// Test if flag is set. Return "on" or "off".
func (f Flag) Get(o Flag) string {
	return bool2string(f&o == o)
}

// Machine information.
type Machine struct {
	Name       string
	UUID       string
	State      MachineState
	CPUs       uint
	Memory     uint // main memory (in MB)
	VRAM       uint // video memory (in MB)
	CfgFile    string
	BaseFolder string
	OSType     string
	Flag       Flag
	BootOrder  []string // max 4 slots, each in {none|floppy|dvd|disk|net}
	NICs       []NIC
}

func New() *Machine {
	return &Machine{
		BootOrder: make([]string, 0, 4),
		NICs:      make([]NIC, 0, 4),
	}
}

// Refresh reloads the machine information.
func (m *Machine) Refresh() error {
	id := m.Name
	if id == "" {
		id = m.UUID
	}
	mm, err := GetMachine(id)
	if err != nil {
		return err
	}
	*m = *mm
	return nil
}

// Start starts the machine.
func (m *Machine) Start() error {
	switch m.State {
	case Paused:
		return vbm("controlvm", m.Name, "resume")
	case Poweroff, Saved, Aborted:
		return vbm("startvm", m.Name, "--type", "headless")
	}
	return nil
}

// Suspend suspends the machine and saves its state to disk.
func (m *Machine) Save() error {
	switch m.State {
	case Paused:
		if err := m.Start(); err != nil {
			return err
		}
	case Poweroff, Aborted, Saved:
		return nil
	}
	return vbm("controlvm", m.Name, "savestate")
}

// Pause pauses the execution of the machine.
func (m *Machine) Pause() error {
	switch m.State {
	case Paused, Poweroff, Aborted, Saved:
		return nil
	}
	return vbm("controlvm", m.Name, "pause")
}

// Stop gracefully stops the machine.
func (m *Machine) Stop() error {
	switch m.State {
	case Poweroff, Aborted, Saved:
		return nil
	case Paused:
		if err := m.Start(); err != nil {
			return err
		}
	}

	/*
	 * Don't perform busy wait here because ACPI power off doesn't have effect in all situations.
	for m.State != Poweroff { // busy wait until the machine is stopped
		if err := vbm("controlvm", m.Name, "acpipowerbutton"); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
		if err := m.Refresh(); err != nil {
			return err
		}
	}
	return nil
	*/
	return vbm("controlvm", m.Name, "acpipowerbutton")
}

// Poweroff forcefully stops the machine. State is lost and might corrupt the disk image.
func (m *Machine) Poweroff() error {
	switch m.State {
	case Poweroff, Aborted, Saved:
		return nil
	}
	return vbm("controlvm", m.Name, "poweroff")
}

// Restart gracefully restarts the machine.
func (m *Machine) Restart() error {
	switch m.State {
	case Paused, Saved:
		if err := m.Start(); err != nil {
			return err
		}
	}
	if err := m.Stop(); err != nil {
		return err
	}
	return m.Start()
}

// Reset forcefully restarts the machine. State is lost and might corrupt the disk image.
func (m *Machine) Reset() error {
	switch m.State {
	case Paused, Saved:
		if err := m.Start(); err != nil {
			return err
		}
	}
	return vbm("controlvm", m.Name, "reset")
}

// Delete deletes the machine and associated disk images.
func (m *Machine) Delete() error {
	if err := m.Poweroff(); err != nil {
		return err
	}
	return vbm("unregistervm", m.Name, "--delete")
}

var mutex sync.Mutex

// GetMachine finds a machine by its name or UUID.
func GetMachine(id string) (*Machine, error) {
	/* There is a strage behavior where running multiple instances of
	'VBoxManage showvminfo' on same VM simultaneously can return an error of
	'object is not ready (E_ACCESSDENIED)', so we sequential the operation with a mutex.
	Note if you are running multiple process of go-virtualbox or 'showvminfo'
	in the command line side by side, this not gonna work. */
	mutex.Lock()
	stdout, stderr, err := vbmOutErr("showvminfo", id, "--machinereadable")
	mutex.Unlock()
	if err != nil {
		if reMachineNotFound.FindString(stderr) != "" {
			return nil, ErrMachineNotExist
		}
		return nil, err
	}

	/* Read all VM info into a map */
	propMap := make(map[string]string)
	s := bufio.NewScanner(strings.NewReader(stdout))
	for s.Scan() {
		res := reVMInfoLine.FindStringSubmatch(s.Text())
		if res == nil {
			continue
		}
		key := res[1]
		if key == "" {
			key = res[2]
		}
		val := res[3]
		if val == "" {
			val = res[4]
		}
		propMap[key] = val
	}

	/* Extract basic info */
	m := New()
	m.Name = propMap["name"]
	m.UUID = propMap["UUID"]
	m.State = MachineState(propMap["VMState"])
	n, err := strconv.ParseUint(propMap["memory"], 10, 32)
	if err != nil {
		return nil, err
	}
	m.Memory = uint(n)
	n, err = strconv.ParseUint(propMap["cpus"], 10, 32)
	if err != nil {
		return nil, err
	}
	m.CPUs = uint(n)
	n, err = strconv.ParseUint(propMap["vram"], 10, 32)
	if err != nil {
		return nil, err
	}
	m.VRAM = uint(n)
	m.CfgFile = propMap["CfgFile"]
	m.BaseFolder = filepath.Dir(m.CfgFile)

	/* Extract NIC info */
	for i := 1; i <= 4; i++ {
		var nic NIC
		nicType, ok := propMap[fmt.Sprintf("nic%d", i)]
		if !ok || nicType == "none" {
			break
		}
		nic.Network = NICNetwork(nicType)
		nic.Hardware = NICHardware(propMap[fmt.Sprintf("nictype%d", i)])
		if nic.Hardware == "" {
			return nil, fmt.Errorf("Could not find corresponding 'nictype%d'", i)
		}
		nic.MacAddr = propMap[fmt.Sprintf("macaddress%d", i)]
		if nic.MacAddr == "" {
			return nil, fmt.Errorf("Could not find corresponding 'macaddress%d'", i)
		}
		if nic.Network == NICNetHostonly {
			nic.HostInterface = propMap[fmt.Sprintf("hostonlyadapter%d", i)]
		} else if nic.Network == NICNetBridged {
			nic.HostInterface = propMap[fmt.Sprintf("bridgeadapter%d", i)]
		}
		m.NICs = append(m.NICs, nic)
	}

	/* Extract Boot Device info */
	for i := 1; i <= 4; i++ {
		dev, ok := propMap[fmt.Sprintf("boot%d", i)]
		if !ok || dev == "none" {
			break
		}

		m.BootOrder = append(m.BootOrder, dev)
	}

	// Extract Flags
	if propMap["acpi"] == "on" {
		m.Flag |= F_acpi
	}
	if propMap["rtcuseutc"] == "on" {
		m.Flag |= F_rtcuseutc
	}
	if propMap["ioapic"] == "on" {
		m.Flag |= F_ioapic
	}
	if propMap["pae"] == "on" {
		m.Flag |= F_pae
	}
	if propMap["longmode"] == "on" {
		m.Flag |= F_longmode
	}
	if propMap["hpet"] == "on" {
		m.Flag |= F_hpet
	}
	if propMap["hwvirtex"] == "on" {
		m.Flag |= F_hwvirtex
	}
	if propMap["triplefaultreset"] == "on" {
		m.Flag |= F_triplefaultreset
	}
	if propMap["nestedpaging"] == "on" {
		m.Flag |= F_nestedpaging
	}
	if propMap["largepages"] == "on" {
		m.Flag |= F_largepages
	}
	if propMap["vtxvpid"] == "on" {
		m.Flag |= F_vtxvpid
	}
	if propMap["vtxux"] == "on" {
		m.Flag |= F_vtxux
	}
	if propMap["accelerate3d"] == "on" {
		m.Flag |= F_accelerate3d
	}


	if err := s.Err(); err != nil {
		return nil, err
	}
	return m, nil
}

// ListMachines lists all registered machines.
func ListMachines() ([]*Machine, error) {
	out, err := vbmOut("list", "vms")
	if err != nil {
		return nil, err
	}
	ms := []*Machine{}
	s := bufio.NewScanner(strings.NewReader(out))
	for s.Scan() {
		res := reVMNameUUID.FindStringSubmatch(s.Text())
		if res == nil {
			continue
		}
		m, err := GetMachine(res[1])
		if err != nil {
			// Sometimes a VM is listed but not available, so we need to handle this.
			if err == ErrMachineNotExist {
				continue
			} else {
				return nil, err
			}
		}
		ms = append(ms, m)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return ms, nil
}

// CreateMachine creates a new machine. If basefolder is empty, use default.
func CreateMachine(name, basefolder string) (*Machine, error) {
	if name == "" {
		return nil, fmt.Errorf("machine name is empty")
	}

	// Check if a machine with the given name already exists.
	ms, err := ListMachines()
	if err != nil {
		return nil, err
	}
	for _, m := range ms {
		if m.Name == name {
			return nil, ErrMachineExist
		}
	}

	// Create and register the machine.
	args := []string{"createvm", "--name", name, "--register"}
	if basefolder != "" {
		args = append(args, "--basefolder", basefolder)
	}
	if err := vbm(args...); err != nil {
		return nil, err
	}

	m, err := GetMachine(name)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Modify changes the settings of the machine.
func (m *Machine) Modify() error {
	args := []string{"modifyvm", m.Name,
		"--firmware", "bios",
		"--bioslogofadein", "off",
		"--bioslogofadeout", "off",
		"--bioslogodisplaytime", "0",
		"--biosbootmenu", "disabled",

		"--cpus", fmt.Sprintf("%d", m.CPUs),
		"--memory", fmt.Sprintf("%d", m.Memory),
		"--vram", fmt.Sprintf("%d", m.VRAM),

		"--acpi", m.Flag.Get(F_acpi),
		"--ioapic", m.Flag.Get(F_ioapic),
		"--rtcuseutc", m.Flag.Get(F_rtcuseutc),
		"--pae", m.Flag.Get(F_pae),
		"--longmode", m.Flag.Get(F_longmode),
		"--hpet", m.Flag.Get(F_hpet),
		"--hwvirtex", m.Flag.Get(F_hwvirtex),
		"--triplefaultreset", m.Flag.Get(F_triplefaultreset),
		"--nestedpaging", m.Flag.Get(F_nestedpaging),
		"--largepages", m.Flag.Get(F_largepages),
		"--vtxvpid", m.Flag.Get(F_vtxvpid),
		"--vtxux", m.Flag.Get(F_vtxux),
		"--accelerate3d", m.Flag.Get(F_accelerate3d),
	}

	for i, dev := range m.BootOrder {
		if i > 3 {
			break // Only four slots `--boot{1,2,3,4}`. Ignore the rest.
		}
		args = append(args, fmt.Sprintf("--boot%d", i+1), dev)
	}

	for i := len(m.BootOrder); i < 4; i++ {
		args = append(args, fmt.Sprintf("--boot%d", i+1), "none")
	}

	for i, nic := range m.NICs {
		n := i + 1
		args = append(args,
			fmt.Sprintf("--nic%d", n), string(nic.Network),
			fmt.Sprintf("--nictype%d", n), string(nic.Hardware),
			fmt.Sprintf("--cableconnected%d", n), "on")
		if nic.Network == NICNetHostonly {
			args = append(args, fmt.Sprintf("--hostonlyadapter%d", n), nic.HostInterface)
		} else if nic.Network == NICNetBridged {
			args = append(args, fmt.Sprintf("--bridgeadapter%d", n), nic.HostInterface)
		}
	}

	fmt.Println(args)

	if err := vbm(args...); err != nil {
		return err
	}
	return m.Refresh()
}

// AddNATPF adds a NAT port forarding rule to the n-th NIC with the given name.
func (m *Machine) AddNATPF(n int, name string, rule PFRule) error {
	return vbm("controlvm", m.Name, fmt.Sprintf("natpf%d", n),
		fmt.Sprintf("%s,%s", name, rule.Format()))
}

// DelNATPF deletes the NAT port forwarding rule with the given name from the n-th NIC.
func (m *Machine) DelNATPF(n int, name string) error {
	return vbm("controlvm", m.Name, fmt.Sprintf("natpf%d", n), "delete", name)
}

// SetNIC set the n-th NIC.
func (m *Machine) SetNIC(n int, nic NIC) error {
	args := []string{"modifyvm", m.Name,
		fmt.Sprintf("--nic%d", n), string(nic.Network),
		fmt.Sprintf("--nictype%d", n), string(nic.Hardware),
		fmt.Sprintf("--cableconnected%d", n), "on",
	}

	if nic.Network == NICNetHostonly {
		args = append(args, fmt.Sprintf("--hostonlyadapter%d", n), nic.HostInterface)
	} else if nic.Network == NICNetBridged {
		args = append(args, fmt.Sprintf("--bridgeadapter%d", n), nic.HostInterface)
	}
	return vbm(args...)
}

// AddStorageCtl adds a storage controller with the given name.
func (m *Machine) AddStorageCtl(name string, ctl StorageController) error {
	args := []string{"storagectl", m.Name, "--name", name}
	if ctl.SysBus != "" {
		args = append(args, "--add", string(ctl.SysBus))
	}
	if ctl.Ports > 0 {
		args = append(args, "--portcount", fmt.Sprintf("%d", ctl.Ports))
	}
	if ctl.Chipset != "" {
		args = append(args, "--controller", string(ctl.Chipset))
	}
	args = append(args, "--hostiocache", bool2string(ctl.HostIOCache))
	args = append(args, "--bootable", bool2string(ctl.Bootable))
	return vbm(args...)
}

// DelStorageCtl deletes the storage controller with the given name.
func (m *Machine) DelStorageCtl(name string) error {
	return vbm("storagectl", m.Name, "--name", name, "--remove")
}

// AttachStorage attaches a storage medium to the named storage controller.
func (m *Machine) AttachStorage(ctlName string, medium StorageMedium) error {
	return vbm("storageattach", m.Name, "--storagectl", ctlName,
		"--port", fmt.Sprintf("%d", medium.Port),
		"--device", fmt.Sprintf("%d", medium.Device),
		"--type", string(medium.DriveType),
		"--medium", medium.Medium,
	)
}

// GetGuestProperty get guest property from the VM, mose of these properties
//	need VirtualBox Guest Addition be installed on the guest.
// Use 'VBoxManage guestproperty enumerate' to list all available properties.
func (m *Machine) GetGuestProperty(key string) (*string, error) {
	value, err := vbmOut("guestproperty", "get", m.Name, key)
	if err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	/* 'guestproperty get' returns 0 even when the key is not found,
	so we need to check stdout for this case */
	if strings.HasPrefix(value, "No value set") {
		return nil, nil
	} else {
		trimmed := strings.TrimPrefix(value, "Value: ")
		return &trimmed, nil
	}
}

// SetExtraData attaches custom string to the VM.
func (m *Machine) SetExtraData(key, val string) error {
	return vbm("setextradata", m.Name, key, val)
}

// SetExtraData retrieves custom string from the VM.
func (m *Machine) GetExtraData(key string) (*string, error) {
	value, err := vbmOut("getextradata", m.Name, key)
	if err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	/* 'getextradata get' returns 0 even when the key is not found,
	so we need to check stdout for this case */
	if strings.HasPrefix(value, "No value set") {
		return nil, nil
	} else {
		trimmed := strings.TrimPrefix(value, "Value: ")
		return &trimmed, nil
	}
}

// SetExtraData removes custom string from the VM.
func (m *Machine) DeleteExtraData(key string) error {
	return vbm("setextradata", m.Name, key)
}
