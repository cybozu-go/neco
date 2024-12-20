package main

type mockNetIF struct {
	addrs  []string
	called []string
	err    error
}

func (n *mockNetIF) Name() string {
	return "mock"
}

func (n *mockNetIF) Up() error {
	n.called = append(n.called, "up")
	return n.err
}

func (n *mockNetIF) Down() error {
	n.called = append(n.called, "down")
	return n.err
}

func (n *mockNetIF) ListAddrs() ([]string, error) {
	n.called = append(n.called, "list")
	if n.err != nil {
		return nil, n.err
	}
	return n.addrs, nil
}

func (n *mockNetIF) AddAddr(addr string) error {
	n.called = append(n.called, "add:"+addr)
	return n.err
}

func (n *mockNetIF) DeleteAddr(addr string) error {
	n.called = append(n.called, "delete:"+addr)
	return n.err
}

func (n *mockNetIF) DeleteAllAddr() error {
	n.called = append(n.called, "deleteAll")
	return n.err
}
