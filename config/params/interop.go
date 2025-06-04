package params
// InteropConfig provides a generic config suitable for interop testing.
func InteropConfig() *BeaconChainConfig {
	c := MainnetConfig().Copy()
// fixme: gluon : change start
    // Custom fork versions
    c.ConfigName = InteropName
    c.GenesisForkVersion = []byte{0x20, 0x00, 0x00, 0x89}
    c.AltairForkVersion = []byte{0x20, 0x00, 0x00, 0x90}
    c.BellatrixForkVersion = []byte{0x20, 0x00, 0x00, 0x91}
    c.CapellaForkVersion = []byte{0x20, 0x00, 0x00, 0x92}
    c.DenebForkVersion = []byte{0x20, 0x00, 0x00, 0x93}

	c.GenesisEpoch =         0
	c.AltairForkEpoch =      0
	c.BellatrixForkEpoch =   0
	c.CapellaForkEpoch =     0
	c.DenebForkEpoch =       18446744073709551615
// fixme: gluon : change end

	c.InitializeForkSchedule()
	return c
}
